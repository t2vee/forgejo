// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package archiver

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/unknwon/com"
)

var queueMutex sync.Mutex

func TestMain(m *testing.M) {
	models.MainTest(m, filepath.Join("..", ".."))
}

func allComplete(inFlight []*ArchiveRequest) bool {
	for _, req := range inFlight {
		if !req.IsComplete() {
			return false
		}
	}

	return true
}

func waitForCount(t *testing.T, num int) {
	var numQueued int

	// Wait for up to 10 seconds for the queue to be impacted.
	timeout := time.Now().Add(10 * time.Second)
	for {
		numQueued = len(archiveInProgress)
		if numQueued == num || time.Now().After(timeout) {
			break
		}
	}

	assert.Equal(t, num, len(archiveInProgress))
}

func releaseOneEntry(t *testing.T, inFlight []*ArchiveRequest) {
	numQueued := len(archiveInProgress)

	// Release one, then WaitForCompletion.  We'll get signalled when ready.
	// This works out to be quick and race-free, as we'll get signalled when the
	// archival goroutine proceeds to dequeue the now-complete archive but we
	// can't pick up the queue lock again until it's done removing it from
	// archiveInProgress.  We'll remain waiting on the queue lock in
	// WaitForCompletion() until we can safely acquire the lock.
	LockQueue()
	archiveQueueReleaseCond.Signal()
	WaitForCompletion()
	UnlockQueue()

	// Also make sure that we released only one.
	assert.Equal(t, numQueued-1, len(archiveInProgress))
}

func TestArchive_Basic(t *testing.T) {
	assert.NoError(t, models.PrepareTestDatabase())

	// Create a new context here, because we may want to use locks or need other
	// initial state here.
	NewContext()
	archiveQueueMutex = &queueMutex
	archiveQueueStartCond = sync.NewCond(&queueMutex)
	archiveQueueReleaseCond = sync.NewCond(&queueMutex)
	defer func() {
		archiveQueueMutex = nil
		archiveQueueStartCond = nil
		archiveQueueReleaseCond = nil
	}()

	ctx := test.MockContext(t, "user27/repo49")
	firstCommit, secondCommit := "51f84af23134", "aacbdfe9e1c4"

	bogusReq := DeriveRequestFrom(ctx, firstCommit+".zip")
	assert.Nil(t, bogusReq)

	test.LoadRepo(t, ctx, 49)
	bogusReq = DeriveRequestFrom(ctx, firstCommit+".zip")
	assert.Nil(t, bogusReq)

	test.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	// Check a series of bogus requests.
	// Step 1, valid commit with a bad extension.
	bogusReq = DeriveRequestFrom(ctx, firstCommit+".dilbert")
	assert.Nil(t, bogusReq)

	// Step 2, missing commit.
	bogusReq = DeriveRequestFrom(ctx, "dbffff.zip")
	assert.Nil(t, bogusReq)

	// Step 3, doesn't look like branch/tag/commit.
	bogusReq = DeriveRequestFrom(ctx, "db.zip")
	assert.Nil(t, bogusReq)

	// Now two valid requests, firstCommit with valid extensions.
	zipReq := DeriveRequestFrom(ctx, firstCommit+".zip")
	assert.NotNil(t, zipReq)

	tgzReq := DeriveRequestFrom(ctx, firstCommit+".tar.gz")
	assert.NotNil(t, tgzReq)

	secondReq := DeriveRequestFrom(ctx, secondCommit+".zip")
	assert.NotNil(t, secondReq)

	inFlight := make([]*ArchiveRequest, 3)
	inFlight[0] = zipReq
	inFlight[1] = tgzReq
	inFlight[2] = secondReq

	ArchiveRepository(zipReq)
	waitForCount(t, 1)
	ArchiveRepository(tgzReq)
	waitForCount(t, 2)
	ArchiveRepository(secondReq)
	waitForCount(t, 3)

	// Make sure sending an unprocessed request through doesn't affect the queue
	// count.
	ArchiveRepository(zipReq)

	// Sleep two seconds to make sure the queue doesn't change.
	time.Sleep(2 * time.Second)
	assert.Equal(t, 3, len(archiveInProgress))

	// Release them all, they'll then stall at the archiveQueueReleaseCond while
	// we examine the queue state.
	queueMutex.Lock()
	archiveQueueStartCond.Broadcast()
	queueMutex.Unlock()

	// 10 second timeout for them all to complete.
	timeout := time.Now().Add(10 * time.Second)
	for {
		if allComplete(inFlight) || time.Now().After(timeout) {
			break
		}
	}

	assert.True(t, zipReq.IsComplete())
	assert.True(t, tgzReq.IsComplete())
	assert.True(t, secondReq.IsComplete())
	assert.True(t, com.IsExist(zipReq.GetArchivePath()))
	assert.True(t, com.IsExist(tgzReq.GetArchivePath()))
	assert.True(t, com.IsExist(secondReq.GetArchivePath()))

	// Queues should not have drained yet, because we haven't released them.
	// Do so now.
	assert.Equal(t, len(archiveInProgress), 3)

	zipReq2 := DeriveRequestFrom(ctx, firstCommit+".zip")
	// This zipReq should match what's sitting in the queue, as we haven't
	// let it release yet.  From the consumer's point of view, this looks like
	// a long-running archive task.
	assert.Equal(t, zipReq, zipReq2)

	// We still have the other three stalled at completion, waiting to remove
	// from archiveInProgress.  Try to submit this new one before its
	// predecessor has cleared out of the queue.
	ArchiveRepository(zipReq2)

	// Make sure the queue hasn't grown any.
	assert.Equal(t, 3, len(archiveInProgress))

	// Make sure the queue drains properly
	releaseOneEntry(t, inFlight)
	assert.Equal(t, 2, len(archiveInProgress))
	releaseOneEntry(t, inFlight)
	assert.Equal(t, 1, len(archiveInProgress))
	releaseOneEntry(t, inFlight)
	assert.Equal(t, 0, len(archiveInProgress))

	zipReq2 = DeriveRequestFrom(ctx, firstCommit+".zip")
	// Now, we're guaranteed to have released the original zipReq from the queue.
	// Ensure that we don't get handed back the released entry somehow, but they
	// should remain functionally equivalent in all fields.
	assert.Equal(t, zipReq, zipReq2)
	assert.False(t, zipReq == zipReq2)

	// Same commit, different compression formats should have different names.
	// Ideally, the extension would match what we originally requested.
	assert.NotEqual(t, zipReq.GetArchiveName(), tgzReq.GetArchiveName())
	assert.NotEqual(t, zipReq.GetArchiveName(), secondReq.GetArchiveName())
}
