// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"

	"code.gitea.io/gitea/models/db"
)

type QuotaUsed struct { //revive:disable-line:exported
	Git struct {
		Code int64 `json:"code"`
		LFS  int64 `json:"lfs"`
	} `json:"git"`
	Assets struct {
		Attachments int64 `json:"attachments"`
		Artifacts   int64 `json:"artifacts"`
		Packages    int64 `json:"packages"`
	} `json:"assets"`
}

func (u *QuotaUsed) TotalSize() int64 {
	return u.GitSize() + u.AssetsSize()
}

func (u *QuotaUsed) GitSize() int64 {
	return u.Git.Code + u.Git.LFS
}

func (u *QuotaUsed) AssetsSize() int64 {
	return u.Assets.Attachments + u.Assets.Packages + u.Assets.Artifacts
}

func (u *QuotaUsed) getUsedForCategory(category QuotaLimitCategory) int64 {
	switch category {
	case QuotaLimitCategoryGitTotal:
		return u.GitSize()
	case QuotaLimitCategoryGitCode:
		return u.Git.Code
	case QuotaLimitCategoryGitLFS:
		return u.Git.LFS

	case QuotaLimitCategoryAssetAttachments:
		return u.Assets.Attachments
	case QuotaLimitCategoryAssetArtifacts:
		return u.Assets.Artifacts
	case QuotaLimitCategoryAssetPackages:
		return u.Assets.Packages

	case QuotaLimitCategoryWiki:
		return 0
	}

	return 0
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

	_, err = db.GetEngine(ctx).Select("SUM(size) AS size").
		Table("attachment").
		Where("uploader_id = ?", userID).
		Get(&used.Assets.Attachments)
	if err != nil {
		return nil, err
	}

	_, err = db.GetEngine(ctx).Select("SUM(file_compressed_size) AS size").
		Table("action_artifact").
		Where("owner_id = ?", userID).
		Get(&used.Assets.Artifacts)
	if err != nil {
		return nil, err
	}

	_, err = db.GetEngine(ctx).Select("SUM(package_blob.size) AS size").
		Table("package_blob").
		Join("INNER", "`package_file`", "`package_file`.blob_id = `package_blob`.id").
		Join("INNER", "`package_version`", "`package_file`.version_id = `package_version`.id").
		Join("INNER", "`package`", "`package_version`.package_id = `package`.id").
		Where("`package`.owner_id = ?", userID).
		Get(&used.Assets.Packages)
	if err != nil {
		return nil, err
	}

	return &used, nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
