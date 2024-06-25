// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"

	"code.gitea.io/gitea/models/db"
)

// QuotaGroup represents a quota group
// swagger::model
type QuotaGroup struct { //revive:disable-line:exported
	ID int64 `json:"-" xorm:"pk autoincr"`
	// Name of the quota group
	Name string `json:"name" xorm:"UNIQUE NOT NULL" binding:"Required"`

	LimitTotal *int64 `json:"limit_total,omitempty"`

	LimitGitTotal *int64 `json:"limit_git_total,omitempty"`
	LimitGitCode  *int64 `json:"limit_git_code,omitempty"`
	LimitGitLFS   *int64 `json:"limit_git_lfs,omitempty"`

	LimitAssetTotal       *int64 `json:"limit_asset_total,omitempty"`
	LimitAssetAttachments *int64 `json:"limit_asset_attachments,omitempty"`
	LimitAssetPackages    *int64 `json:"limit_asset_packages,omitempty"`
	LimitAssetArtifacts   *int64 `json:"limit_asset_artifacts,omitempty"`
}

// QuotaGroupList is a list of quota groups
// swagger:model
type QuotaGroupList []*QuotaGroup //revive:disable-line:exported

func (qg *QuotaGroup) AddUserByID(ctx context.Context, userID int64) error {
	_, err := db.GetEngine(ctx).Insert(&QuotaMapping{
		Kind:         QuotaKindUser,
		MappedID:     userID,
		QuotaGroupID: qg.ID,
	})
	return err
}

func (qg *QuotaGroup) RemoveUserByID(ctx context.Context, userID int64) error {
	_, err := db.GetEngine(ctx).Where("kind = ? AND mapped_id = ?", QuotaKindUser, userID).Delete(QuotaMapping{})
	return err
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
