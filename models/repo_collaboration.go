// Copyright 2016 The Gogs Authors. All rights reserved.
// Copyright 2020 The Gitea Authors.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/perm"
	access_model "code.gitea.io/gitea/models/perm/access"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"

	"xorm.io/builder"
)

func addCollaborator(ctx context.Context, repo *repo_model.Repository, u *user_model.User) error {
	collaboration := &repo_model.Collaboration{
		RepoID: repo.ID,
		UserID: u.ID,
	}

	has, err := db.GetByBean(ctx, collaboration)
	if err != nil {
		return err
	} else if has {
		return nil
	}
	collaboration.Mode = perm.AccessModeWrite

	if err = db.Insert(ctx, collaboration); err != nil {
		return err
	}

	return access_model.RecalculateUserAccess(ctx, repo, u.ID)
}

// AddCollaborator adds new collaboration to a repository with default access mode.
func AddCollaborator(repo *repo_model.Repository, u *user_model.User) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()

	if err := addCollaborator(ctx, repo, u); err != nil {
		return err
	}

	return committer.Commit()
}

// DeleteCollaboration removes collaboration relation between the user and repository.
func DeleteCollaboration(repo *repo_model.Repository, uid int64) (err error) {
	collaboration := &repo_model.Collaboration{
		RepoID: repo.ID,
		UserID: uid,
	}

	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()

	if has, err := db.GetEngine(ctx).Delete(collaboration); err != nil || has == 0 {
		return err
	} else if err = access_model.RecalculateAccesses(ctx, repo); err != nil {
		return err
	}

	if err = repo_model.WatchRepo(ctx, uid, repo.ID, false); err != nil {
		return err
	}

	if err = reconsiderWatches(ctx, repo, uid); err != nil {
		return err
	}

	// Unassign a user from any issue (s)he has been assigned to in the repository
	if err := reconsiderRepoIssuesAssignee(ctx, repo, uid); err != nil {
		return err
	}

	return committer.Commit()
}

func reconsiderRepoIssuesAssignee(ctx context.Context, repo *repo_model.Repository, uid int64) error {
	user, err := user_model.GetUserByIDCtx(ctx, uid)
	if err != nil {
		return err
	}

	if canAssigned, err := access_model.CanBeAssigned(ctx, user, repo, true); err != nil || canAssigned {
		return err
	}

	if _, err := db.GetEngine(ctx).Where(builder.Eq{"assignee_id": uid}).
		In("issue_id", builder.Select("id").From("issue").Where(builder.Eq{"repo_id": repo.ID})).
		Delete(&IssueAssignees{}); err != nil {
		return fmt.Errorf("Could not delete assignee[%d] %v", uid, err)
	}
	return nil
}

func reconsiderWatches(ctx context.Context, repo *repo_model.Repository, uid int64) error {
	if has, err := access_model.HasAccess(ctx, uid, repo); err != nil || has {
		return err
	}
	if err := repo_model.WatchRepo(ctx, uid, repo.ID, false); err != nil {
		return err
	}

	// Remove all IssueWatches a user has subscribed to in the repository
	return removeIssueWatchersByRepoID(ctx, uid, repo.ID)
}
