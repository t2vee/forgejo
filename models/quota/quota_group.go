// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"

	"xorm.io/builder"
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

func ListQuotaGroups(ctx context.Context) ([]*QuotaGroup, error) {
	var groups []*QuotaGroup
	err := db.GetEngine(ctx).Find(&groups)
	return groups, err
}

func CreateQuotaGroup(ctx context.Context, group QuotaGroup) error {
	_, err := db.GetEngine(ctx).Insert(group)
	return err
}

func ListUsersInQuotaGroup(ctx context.Context, name string) ([]*user_model.User, error) {
	group, err := GetQuotaGroupByName(ctx, name)
	if err != nil {
		return nil, err
	}

	var users []*user_model.User
	err = db.GetEngine(ctx).Select("`user`.*").
		Table("user").
		Join("INNER", "`quota_mapping`", "`quota_mapping`.mapped_id = `user`.id").
		Where("`quota_mapping`.kind = ? AND `quota_mapping`.quota_group_id = ?", QuotaKindUser, group.ID).
		Find(&users)
	return users, err
}

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

func GetQuotaGroupByName(ctx context.Context, name string) (*QuotaGroup, error) {
	var group QuotaGroup
	has, err := db.GetEngine(ctx).Where("name = ?", name).Get(&group)
	if has {
		return &group, nil
	}
	return nil, err
}

func IsQuotaGroupInUse(ctx context.Context, name string) bool {
	var inuse bool

	group, err := GetQuotaGroupByName(ctx, name)
	if err != nil || group == nil {
		return false
	}

	_, err = db.GetEngine(ctx).Select("true").
		Table("quota_mapping").
		Where("`quota_mapping`.quota_group_id = ?", group.ID).
		Get(&inuse)
	if err != nil {
		return false
	}
	return inuse
}

func DeleteQuotaGroupByName(ctx context.Context, name string) error {
	_, err := db.GetEngine(ctx).Where("name = ?", name).Delete(QuotaGroup{})
	return err
}

func GetQuotaGroupsForUser(ctx context.Context, userID int64) ([]*QuotaGroup, error) {
	var groups []*QuotaGroup
	err := db.GetEngine(ctx).
		Where(builder.In("id",
			builder.Select("quota_group_id").
				From("quota_mapping").
				Where(builder.And(
					builder.Eq{"kind": QuotaKindUser},
					builder.Eq{"mapped_id": userID}),
				))).
		Find(&groups)
	if err != nil {
		return nil, err
	}

	if len(groups) == 0 {
		err = db.GetEngine(ctx).Where(builder.In("name", setting.Quota.DefaultGroups)).Find(&groups)
		if err != nil {
			return nil, err
		}
		if len(groups) == 0 {
			return nil, nil
		}
	}

	return groups, nil
}
