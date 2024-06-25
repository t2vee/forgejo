// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"

	"code.gitea.io/gitea/models/db"
	quota_model "code.gitea.io/gitea/models/quota"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"

	"xorm.io/builder"
)

func ListQuotaGroups(ctx context.Context) ([]*quota_model.QuotaGroup, error) {
	var groups []*quota_model.QuotaGroup
	err := db.GetEngine(ctx).Find(&groups)
	return groups, err
}

func CreateQuotaGroup(ctx context.Context, group quota_model.QuotaGroup) error {
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
		Where("`quota_mapping`.kind = ? AND `quota_mapping`.quota_group_id = ?", quota_model.QuotaKindUser, group.ID).
		Find(&users)
	return users, err
}

func GetQuotaGroupByName(ctx context.Context, name string) (*quota_model.QuotaGroup, error) {
	var group quota_model.QuotaGroup
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
	_, err := db.GetEngine(ctx).Where("name = ?", name).Delete(quota_model.QuotaGroup{})
	return err
}

func GetQuotaGroupsForUser(ctx context.Context, userID int64) ([]*quota_model.QuotaGroup, error) {
	var groups []*quota_model.QuotaGroup
	err := db.GetEngine(ctx).
		Where(builder.In("id",
			builder.Select("quota_group_id").
				From("quota_mapping").
				Where(builder.And(
					builder.Eq{"kind": quota_model.QuotaKindUser},
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
