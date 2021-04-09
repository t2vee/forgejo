// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package mirror

import (
	"context"
	"errors"
	"strconv"
	"time"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"
)

// AddPushMirrorRemote registers the push mirror remote.
func AddPushMirrorRemote(m *models.PushMirror, addr string) error {
	addRemoteAndConfig := func(addr, path string) error {
		if _, err := git.NewCommand("remote", "add", "--mirror=push", m.RemoteName, addr).RunInDir(path); err != nil {
			return err
		}
		if _, err := git.NewCommand("config", "--add", "remote."+m.RemoteName+".push", "+refs/heads/*:refs/heads/*").RunInDir(path); err != nil {
			return err
		}
		if _, err := git.NewCommand("config", "--add", "remote."+m.RemoteName+".push", "+refs/tags/*:refs/tags/*").RunInDir(path); err != nil {
			return err
		}
		return nil
	}

	if err := addRemoteAndConfig(addr, m.Repo.RepoPath()); err != nil {
		return err
	}

	if m.Repo.HasWiki() {
		wikiRemoteURL := repository.WikiRemoteURL(addr)
		if len(wikiRemoteURL) > 0 {
			if err := addRemoteAndConfig(wikiRemoteURL, m.Repo.WikiPath()); err != nil {
				return err
			}
		}
	}

	return nil
}

// RemovePushMirrorRemote removes the push mirror remote.
func RemovePushMirrorRemote(m *models.PushMirror) error {
	cmd := git.NewCommand("remote", "rm", m.RemoteName)

	if _, err := cmd.RunInDir(m.Repo.RepoPath()); err != nil {
		return err
	}

	if m.Repo.HasWiki() {
		if _, err := cmd.RunInDir(m.Repo.WikiPath()); err != nil {
			// The wiki remote may not exist
			log.Warn("Wiki Remote[%d] could not be removed: %v", m.ID, err)
		}
	}

	return nil
}

func syncPushMirror(ctx context.Context, mirrorID string) {
	log.Trace("SyncPushMirror [mirror: %s]", mirrorID)
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		// There was a panic whilst syncPushMirror...
		log.Error("PANIC whilst syncPushMirror[%s] Panic: %v\nStacktrace: %s", mirrorID, err, log.Stack(2))
	}()

	id, _ := strconv.ParseInt(mirrorID, 10, 64)
	m, err := models.GetPushMirrorByID(id)
	if err != nil {
		log.Error("GetPushMirrorByID [%s]: %v", mirrorID, err)
		return
	}

	m.UpdatedUnix = timeutil.TimeStampNow()

	log.Trace("SyncPushMirror [mirror: %s][repo: %-v]: Running Sync", mirrorID, m.Repo)
	err = runPushSync(ctx, m)
	if err != nil {
		m.LastError = err.Error()
	}

	log.Trace("SyncPushMirror [mirror: %s][repo: %-v]: Scheduling next update", mirrorID, m.Repo)
	m.ScheduleNextUpdate()
	if err = models.UpdatePushMirror(m); err != nil {
		log.Error("UpdatePushMirror [%s]: %v", mirrorID, err)
	}

	log.Trace("SyncPushMirror [mirror: %s][repo: %-v]: Finished", mirrorID, m.Repo)
}

func runPushSync(ctx context.Context, m *models.PushMirror) error {
	timeout := time.Duration(setting.Git.Timeout.Mirror) * time.Second

	performPush := func(path string) error {
		log.Trace("Pushing %s mirror[%d] remote %s", path, m.ID, m.RemoteName)

		if err := git.Push(path, git.PushOptions{
			Remote:  m.RemoteName,
			Force:   true,
			Mirror:  true,
			Timeout: timeout,
		}); err != nil {
			log.Error("Error pushing %s mirror[%d] remote %s: %v", path, m.ID, m.RemoteName, err)

			remoteAddr, remoteErr := git.GetRemoteAddress(path, m.RemoteName)
			if remoteErr != nil {
				log.Error("GetRemoteAddress(%s) Error %v", path, remoteErr)
				return errors.New("Unexpected error")
			}
			return util.NewURLSanitizedError(err, remoteAddr, true)
		}
		return nil
	}

	err := performPush(m.Repo.RepoPath())
	if err != nil {
		return err
	}

	// TODO LFS

	if m.Repo.HasWiki() {
		err := performPush(m.Repo.WikiPath())
		if err != nil {
			return err
		}
	}

	return nil
}
