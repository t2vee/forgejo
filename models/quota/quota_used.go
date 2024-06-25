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
	GitCode int64
	GitLFS  int64

	AssetAttachments int64
	AssetPackages    int64
	AssetArtifacts   int64
}

func (u *QuotaUsed) Total() int64 {
	return u.Git() + u.Assets()
}

func (u *QuotaUsed) Git() int64 {
	return u.GitCode + u.GitLFS
}

func (u *QuotaUsed) Assets() int64 {
	return u.AssetAttachments + u.AssetPackages + u.AssetArtifacts
}

func (u *QuotaUsed) getUsedForCategory(category QuotaLimitCategory) int64 {
	switch category {
	case QuotaLimitCategoryGitTotal:
		return u.Git()
	case QuotaLimitCategoryGitCode:
		return u.GitCode
	case QuotaLimitCategoryGitLFS:
		return u.GitLFS

	case QuotaLimitCategoryAssetAttachments:
		return u.AssetAttachments
	case QuotaLimitCategoryAssetArtifacts:
		return u.AssetArtifacts
	case QuotaLimitCategoryAssetPackages:
		return u.AssetPackages

	case QuotaLimitCategoryWiki:
		return 0
	}

	return 0
}

func GetQuotaUsedForUser(ctx context.Context, userID int64) (*QuotaUsed, error) {
	type gitSizes struct {
		GitCode int64
		GitLFS  int64
	}
	var gitUsed gitSizes
	_, err := db.GetEngine(ctx).Select("SUM(git_size) AS git_code, SUM(lfs_size) AS git_lfs").
		Table("repository").
		Where("owner_id = ?", userID).
		Get(&gitUsed)
	if err != nil {
		return nil, err
	}

	var attachmentSize int64
	_, err = db.GetEngine(ctx).Select("SUM(size) AS size").
		Table("attachment").
		Where("uploader_id = ?", userID).
		Get(&attachmentSize)
	if err != nil {
		return nil, err
	}

	var artifactSize int64
	_, err = db.GetEngine(ctx).Select("SUM(file_compressed_size) AS size").
		Table("action_artifact").
		Where("owner_id = ?", userID).
		Get(&artifactSize)
	if err != nil {
		return nil, err
	}

	var packageSize int64
	_, err = db.GetEngine(ctx).Select("SUM(package_blob.size) AS size").
		Table("package_blob").
		Join("INNER", "`package_file`", "`package_file`.blob_id = `package_blob`.id").
		Join("INNER", "`package_version`", "`package_file`.version_id = `package_version`.id").
		Join("INNER", "`package`", "`package_version`.package_id = `package`.id").
		Where("`package`.owner_id = ?", userID).
		Get(&packageSize)
	if err != nil {
		return nil, err
	}

	return &QuotaUsed{
		GitCode:          gitUsed.GitCode,
		GitLFS:           gitUsed.GitLFS,
		AssetAttachments: attachmentSize,
		AssetArtifacts:   artifactSize,
		AssetPackages:    packageSize,
	}, nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
