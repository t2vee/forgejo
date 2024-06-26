// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
)

// QuotaUsed represents the quota used by a user
// swagger:model
type QuotaUsed struct { //revive:disable-line:exported
	// Git storage used by the user
	Git struct {
		// Git storage used by the user
		Code int64 `json:"code"`
		// Git LFS storage used by the user
		LFS int64 `json:"lfs"`
	} `json:"git"`
	// Space used by the user for various assets
	Assets struct {
		// Space used by the user's attachments
		Attachments struct {
			// Space used by the user's release attachments
			Releases int64 `json:"releases"`
			// Space used by the user's issue & comment attachments
			Issues int64 `json:"issues"`
		} `json:"attachments"`
		// Space used by the user's artifacts
		Artifacts int64 `json:"artifacts"`
		// Space used by the user's packages
		Packages int64 `json:"packages"`
	} `json:"assets"`
}

func (u *QuotaUsed) TotalSize() int64 {
	return u.GitSize() + u.AssetsSize()
}

func (u *QuotaUsed) GitSize() int64 {
	return u.Git.Code + u.Git.LFS
}

func (u *QuotaUsed) AssetsSize() int64 {
	return u.Assets.Attachments.Releases + u.Assets.Attachments.Issues + u.Assets.Packages + u.Assets.Artifacts
}

func (u *QuotaUsed) getUsedForCategory(category QuotaLimitCategory) int64 {
	switch category {
	case QuotaLimitCategoryGitTotal:
		return u.GitSize()
	case QuotaLimitCategoryGitCode:
		return u.Git.Code
	case QuotaLimitCategoryGitLFS:
		return u.Git.LFS

	case QuotaLimitCategoryAssetAttachmentsReleases:
		return u.Assets.Attachments.Releases
	case QuotaLimitCategoryAssetAttachmentsIssues:
		return u.Assets.Attachments.Issues
	case QuotaLimitCategoryAssetArtifacts:
		return u.Assets.Artifacts
	case QuotaLimitCategoryAssetPackages:
		return u.Assets.Packages

	case QuotaLimitCategoryWiki:
		return 0
	}

	return 0
}

func GetQuotaAttachmentsForUser(ctx context.Context, userID int64, opts db.ListOptions) (int64, *[]*repo_model.Attachment, error) {
	var attachments []*repo_model.Attachment

	sess := db.GetEngine(ctx).
		Table("attachment").
		Join("INNER", "`repository`", "`attachment`.repo_id = `repository`.id").
		Where("`repository`.owner_id = ?", userID)

	count, err := sess.
		Count(new(repo_model.Attachment))
	if err != nil {
		return 0, nil, err
	}

	if opts.PageSize > 0 {
		sess = sess.Limit(opts.PageSize, (opts.Page-1)*opts.PageSize)
	}
	err = sess.OrderBy("`attachment`.size DESC").Find(&attachments)
	if err != nil {
		return 0, nil, err
	}

	return count, &attachments, nil
}

func GetQuotaUsedForUser(ctx context.Context, userID int64) (*QuotaUsed, error) {
	var used QuotaUsed

	_, err := db.GetEngine(ctx).Select("SUM(git_size) AS code, SUM(lfs_size) AS lfs").
		Table("repository").
		Where("owner_id = ?", userID).
		Get(&used.Git)
	if err != nil {
		return nil, err
	}

	_, err = db.GetEngine(ctx).Select("SUM(`attachment`.size) AS size").
		Table("attachment").
		Join("INNER", "`repository`", "`attachment`.repo_id = `repository`.id").
		Where("`repository`.owner_id = ? AND `attachment`.release_id != 0", userID).
		Get(&used.Assets.Attachments.Releases)
	if err != nil {
		return nil, err
	}

	_, err = db.GetEngine(ctx).Select("SUM(`attachment`.size) AS size").
		Table("attachment").
		Join("INNER", "`repository`", "`attachment`.repo_id = `repository`.id").
		Where("`repository`.owner_id = ? AND `attachment`.release_id = 0", userID).
		Get(&used.Assets.Attachments.Issues)
	if err != nil {
		return nil, err
	}

	_, err = db.GetEngine(ctx).Select("SUM(file_compressed_size) AS size").
		Table("action_artifact").
		Join("INNER", "`repository`", "`action_artifact`.repo_id = `repository`.id").
		Where("`repository`.owner_id = ?", userID).
		Get(&used.Assets.Artifacts)
	if err != nil {
		return nil, err
	}

	var floatingPackages int64
	_, err = db.GetEngine(ctx).Select("SUM(package_blob.size) AS size").
		Table("package_blob").
		Join("INNER", "`package_file`", "`package_file`.blob_id = `package_blob`.id").
		Join("INNER", "`package_version`", "`package_file`.version_id = `package_version`.id").
		Join("INNER", "`package`", "`package_version`.package_id = `package`.id").
		Where("`package`.owner_id = ? AND `package`.repo_id = 0", userID).
		Get(&floatingPackages)
	if err != nil {
		return nil, err
	}

	var repoPackages int64
	_, err = db.GetEngine(ctx).Select("SUM(package_blob.size) AS size").
		Table("package_blob").
		Join("INNER", "`package_file`", "`package_file`.blob_id = `package_blob`.id").
		Join("INNER", "`package_version`", "`package_file`.version_id = `package_version`.id").
		Join("INNER", "`package`", "`package_version`.package_id = `package`.id").
		Join("INNER", "`repository`", "`package`.repo_id = `repository`.id").
		Where("`repository`.owner_id = ?", userID).
		Get(&repoPackages)
	if err != nil {
		return nil, err
	}
	used.Assets.Packages = floatingPackages + repoPackages

	return &used, nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
