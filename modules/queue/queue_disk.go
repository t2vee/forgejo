// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"code.gitea.io/gitea/modules/log"

	"gitea.com/lunny/levelqueue"
)

// LevelQueueType is the type for level queue
const LevelQueueType Type = "level"

// LevelQueueConfiguration is the configuration for a LevelQueue
type LevelQueueConfiguration struct {
	WorkerPoolConfiguration
	DataDir string
	Workers int
	Name    string
}

// LevelQueue implements a disk library queue
type LevelQueue struct {
	pool       *WorkerPool
	queue      *levelqueue.Queue
	closed     chan struct{}
	terminated chan struct{}
	lock       sync.Mutex
	exemplar   interface{}
	workers    int
	name       string
}

// NewLevelQueue creates a ledis local queue
func NewLevelQueue(handle HandlerFunc, cfg, exemplar interface{}) (Queue, error) {
	configInterface, err := toConfig(LevelQueueConfiguration{}, cfg)
	if err != nil {
		return nil, err
	}
	config := configInterface.(LevelQueueConfiguration)

	internal, err := levelqueue.Open(config.DataDir)
	if err != nil {
		return nil, err
	}

	queue := &LevelQueue{
		pool:       NewWorkerPool(handle, config.WorkerPoolConfiguration),
		queue:      internal,
		exemplar:   exemplar,
		closed:     make(chan struct{}),
		terminated: make(chan struct{}),
		workers:    config.Workers,
		name:       config.Name,
	}
	queue.pool.qid = GetManager().Add(config.Name, LevelQueueType, config, exemplar, queue.pool)
	return queue, nil
}

// Run starts to run the queue
func (l *LevelQueue) Run(atShutdown, atTerminate func(context.Context, func())) {
	atShutdown(context.Background(), l.Shutdown)
	atTerminate(context.Background(), l.Terminate)
	log.Debug("LevelQueue: %s Starting", l.name)

	go func() {
		_ = l.pool.AddWorkers(l.workers, 0)
	}()

	go l.readToChan()

	log.Trace("LevelQueue: %s Waiting til closed", l.name)
	<-l.closed

	log.Trace("LevelQueue: %s Waiting til done", l.name)
	l.pool.Wait()

	log.Trace("LevelQueue: %s Waiting til cleaned", l.name)
	ctx, cancel := context.WithCancel(context.Background())
	atTerminate(ctx, cancel)
	l.pool.CleanUp(ctx)
	cancel()
	log.Trace("LevelQueue: %s Cleaned", l.name)

}

func (l *LevelQueue) readToChan() {
	for {
		select {
		case <-l.closed:
			// tell the pool to shutdown.
			l.pool.cancel()
			return
		default:
			bs, err := l.queue.RPop()
			if err != nil {
				if err != levelqueue.ErrNotFound {
					log.Error("LevelQueue: %s Error on RPop: %v", l.name, err)
				}
				time.Sleep(time.Millisecond * 100)
				continue
			}

			if len(bs) == 0 {
				time.Sleep(time.Millisecond * 100)
				continue
			}

			data, err := unmarshalAs(bs, l.exemplar)
			if err != nil {
				log.Error("LevelQueue: %s Failed to unmarshal with error: %v", l.name, err)
				time.Sleep(time.Millisecond * 100)
				continue
			}

			log.Trace("LevelQueue %s: Task found: %#v", l.name, data)
			l.pool.Push(data)

		}
	}
}

// Push will push the indexer data to queue
func (l *LevelQueue) Push(data Data) error {
	if !assignableTo(data, l.exemplar) {
		return fmt.Errorf("Unable to assign data: %v to same type as exemplar: %v in %s", data, l.exemplar, l.name)
	}
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return l.queue.LPush(bs)
}

// Flush flushes the queue and blocks till the queue is empty
func (l *LevelQueue) Flush(timeout time.Duration) error {
	return l.pool.Flush(timeout)
}

// Shutdown this queue and stop processing
func (l *LevelQueue) Shutdown() {
	l.lock.Lock()
	defer l.lock.Unlock()
	log.Trace("LevelQueue: %s Shutting down", l.name)
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	log.Debug("LevelQueue: %s Shutdown", l.name)
}

// Terminate this queue and close the queue
func (l *LevelQueue) Terminate() {
	log.Trace("LevelQueue: %s Terminating", l.name)
	l.Shutdown()
	l.lock.Lock()
	select {
	case <-l.terminated:
		l.lock.Unlock()
	default:
		close(l.terminated)
		l.lock.Unlock()
		if err := l.queue.Close(); err != nil && err.Error() != "leveldb: closed" {
			log.Error("Error whilst closing internal queue in %s: %v", l.name, err)
		}

	}
	log.Debug("LevelQueue: %s Terminated", l.name)
}

// Name returns the name of this queue
func (l *LevelQueue) Name() string {
	return l.name
}

func init() {
	queuesMap[LevelQueueType] = NewLevelQueue
}
