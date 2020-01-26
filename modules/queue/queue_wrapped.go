// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.gitea.io/gitea/modules/log"
)

// WrappedQueueType is the type for a wrapped delayed starting queue
const WrappedQueueType Type = "wrapped"

// WrappedQueueConfiguration is the configuration for a WrappedQueue
type WrappedQueueConfiguration struct {
	Underlying  Type
	Timeout     time.Duration
	MaxAttempts int
	Config      interface{}
	QueueLength int
	Name        string
}

type delayedStarter struct {
	internal    Queue
	underlying  Type
	cfg         interface{}
	timeout     time.Duration
	maxAttempts int
	name        string
}

// setInternal must be called with the lock locked.
func (q *delayedStarter) setInternal(atShutdown func(context.Context, func()), handle HandlerFunc, exemplar interface{}) error {
	var ctx context.Context
	var cancel context.CancelFunc
	if q.timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), q.timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	defer cancel()
	// Ensure we also stop at shutdown
	atShutdown(ctx, func() {
		cancel()
	})

	i := 1
	for q.internal == nil {
		select {
		case <-ctx.Done():
			return fmt.Errorf("Timedout creating queue %v with cfg %s in %s", q.underlying, q.cfg, q.name)
		default:
			queue, err := NewQueue(q.underlying, handle, q.cfg, exemplar)
			if err == nil {
				q.internal = queue
				break
			}
			if err.Error() != "resource temporarily unavailable" {
				log.Warn("[Attempt: %d] Failed to create queue: %v for %s cfg: %s error: %v", i, q.underlying, q.name, q.cfg, err)
			}
			i++
			if q.maxAttempts > 0 && i > q.maxAttempts {
				return fmt.Errorf("Unable to create queue %v for %s with cfg %s by max attempts: error: %v", q.underlying, q.name, q.cfg, err)
			}
			sleepTime := 100 * time.Millisecond
			if q.timeout > 0 && q.maxAttempts > 0 {
				sleepTime = (q.timeout - 200*time.Millisecond) / time.Duration(q.maxAttempts)
			}
			t := time.NewTimer(sleepTime)
			select {
			case <-ctx.Done():
				t.Stop()
			case <-t.C:
			}
		}
	}
	return nil
}

// WrappedQueue wraps a delayed starting queue
type WrappedQueue struct {
	delayedStarter
	lock     sync.Mutex
	handle   HandlerFunc
	exemplar interface{}
	channel  chan Data
}

// NewWrappedQueue will attempt to create a queue of the provided type,
// but if there is a problem creating this queue it will instead create
// a WrappedQueue with delayed startup of the queue instead and a
// channel which will be redirected to the queue
func NewWrappedQueue(handle HandlerFunc, cfg, exemplar interface{}) (Queue, error) {
	configInterface, err := toConfig(WrappedQueueConfiguration{}, cfg)
	if err != nil {
		return nil, err
	}
	config := configInterface.(WrappedQueueConfiguration)

	queue, err := NewQueue(config.Underlying, handle, config.Config, exemplar)
	if err == nil {
		// Just return the queue there is no need to wrap
		return queue, nil
	}
	if IsErrInvalidConfiguration(err) {
		// Retrying ain't gonna make this any better...
		return nil, ErrInvalidConfiguration{cfg: cfg}
	}

	queue = &WrappedQueue{
		handle:   handle,
		channel:  make(chan Data, config.QueueLength),
		exemplar: exemplar,
		delayedStarter: delayedStarter{
			cfg:         config.Config,
			underlying:  config.Underlying,
			timeout:     config.Timeout,
			maxAttempts: config.MaxAttempts,
			name:        config.Name,
		},
	}
	_ = GetManager().Add(config.Name, WrappedQueueType, config, exemplar, nil)
	return queue, nil
}

// Name returns the name of the queue
func (q *WrappedQueue) Name() string {
	return q.name + "-wrapper"
}

// Push will push the data to the internal channel checking it against the exemplar
func (q *WrappedQueue) Push(data Data) error {
	if !assignableTo(data, q.exemplar) {
		return fmt.Errorf("unable to assign data: %v to same type as exemplar: %v in %s", data, q.exemplar, q.name)
	}
	q.channel <- data
	return nil
}

func (q *WrappedQueue) flushInternal(timeout time.Duration) error {
	q.lock.Lock()
	if q.internal == nil {
		q.lock.Unlock()
		return fmt.Errorf("not ready to flush wrapped queue %s yet", q.Name())
	}
	q.lock.Unlock()
	return q.internal.Flush(timeout)
}

// Flush flushes the queue and blocks till the queue is empty
func (q *WrappedQueue) Flush(timeout time.Duration) error {
	if timeout < 0 {
		return q.flushInternal(timeout)
	}

	timer := time.NewTimer(timeout)
	errChan := make(chan error, 1)
	go func() {
		errChan <- q.flushInternal(timeout)
		close(errChan)
	}()
	select {
	case err := <-errChan:
		if !timer.Stop() {
			<-timer.C
		}
		return err
	case <-timer.C:
		go func() {
			<-errChan
		}()
		timer.Stop()
		return fmt.Errorf("timed out whilst trying to flush %s", q.Name())
	}
}

// Run starts to run the queue and attempts to create the internal queue
func (q *WrappedQueue) Run(atShutdown, atTerminate func(context.Context, func())) {
	log.Debug("WrappedQueue: %s Starting", q.name)
	q.lock.Lock()
	if q.internal == nil {
		err := q.setInternal(atShutdown, q.handle, q.exemplar)
		q.lock.Unlock()
		if err != nil {
			log.Fatal("Unable to set the internal queue for %s Error: %v", q.Name(), err)
			return
		}
		go func() {
			for data := range q.channel {
				_ = q.internal.Push(data)
			}
		}()
	} else {
		q.lock.Unlock()
	}

	q.internal.Run(atShutdown, atTerminate)
	log.Trace("WrappedQueue: %s Done", q.name)
}

// Shutdown this queue and stop processing
func (q *WrappedQueue) Shutdown() {
	log.Trace("WrappedQueue: %s Shutting down", q.name)
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.internal == nil {
		return
	}
	if shutdownable, ok := q.internal.(Shutdownable); ok {
		shutdownable.Shutdown()
	}
	log.Debug("WrappedQueue: %s Shutdown", q.name)
}

// Terminate this queue and close the queue
func (q *WrappedQueue) Terminate() {
	log.Trace("WrappedQueue: %s Terminating", q.name)
	q.lock.Lock()
	defer q.lock.Unlock()
	if q.internal == nil {
		return
	}
	if shutdownable, ok := q.internal.(Shutdownable); ok {
		shutdownable.Terminate()
	}
	log.Debug("WrappedQueue: %s Terminated", q.name)
}

func init() {
	queuesMap[WrappedQueueType] = NewWrappedQueue
}
