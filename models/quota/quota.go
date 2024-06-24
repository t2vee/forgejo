// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package quota

import (
	"context"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
)

type QuotaKind int //revive:disable-line:exported

const (
	QuotaKindUser QuotaKind = iota
)

type QuotaGroup struct { //revive:disable-line:exported
	ID         int64  `xorm:"pk autoincr"`
	Name       string `xorm:"UNIQUE NOT NULL"`
	LimitGit   int64
	LimitFiles int64
}

type QuotaMapping struct { //revive:disable-line:exported
	ID           int64 `xorm:"pk autoincr"`
	Kind         QuotaKind
	MappedID     int64
	QuotaGroupID int64
}

type QuotaLimits struct { //revive:disable-line:exported
	LimitGit   int64
	LimitFiles int64
}

func init() {
	db.RegisterModel(new(QuotaGroup))
	db.RegisterModel(new(QuotaMapping))
}

func ListQuotaGroups(ctx context.Context) ([]*QuotaGroup, error) {
	var groups []*QuotaGroup
	err := db.GetEngine(ctx).Find(&groups)
	return groups, err
}

func CreateQuotaGroup(ctx context.Context, opts api.CreateQuotaGroupOption) error {
	group := QuotaGroup{
		Name:       opts.Name,
		LimitGit:   opts.LimitGit,
		LimitFiles: opts.LimitFiles,
	}
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

func GetQuotaGroupForUser(ctx context.Context, userID int64) (*QuotaGroup, error) {
	var group QuotaGroup
	has, err := db.GetEngine(ctx).Select("`quota_group`.*").
		Table("quota_group").
		Join("INNER", "`quota_mapping`", "`quota_mapping`.mapped_id = `quota_group`.id").
		Where("`quota_mapping`.kind = ? AND `quota_mapping`.mapped_id = ?", QuotaKindUser, userID).
		Get(&group)
	if err != nil {
		return nil, err
	}

	if !has {
		has, err = db.GetEngine(ctx).Where("name = ?", setting.Quota.DefaultGroup).Get(&group)
		if err != nil {
			return nil, err
		}
		if !has {
			return nil, nil
		}
	}

	return &group, nil
}

func GetQuotaLimitsForUser(ctx context.Context, userID int64) (*QuotaLimits, error) {
	group, err := GetQuotaGroupForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	limits := QuotaLimits{
		LimitGit:   -1,
		LimitFiles: -1,
	}
	if group != nil {
		limits = QuotaLimits{
			LimitGit:   group.LimitGit,
			LimitFiles: group.LimitFiles,
		}
	}
	return &limits, nil
}

func GetGitUseForUser(ctx context.Context, userID int64) (int64, error) {
	var size int64
	_, err := db.GetEngine(ctx).Select("SUM(size) AS size").
		Table("repository").
		Where("owner_id = ?", userID).
		Get(&size)
	return size, err
}

func GetFilesUseForUser(ctx context.Context, userID int64) (int64, error) {
	var totalSize int64
	var size int64

	_, err := db.GetEngine(ctx).Select("SUM(size) AS size").
		Table("attachment").
		Where("uploader_id = ?", userID).
		Get(&size)
	if err != nil {
		return 0, err
	}
	totalSize += size

	size = 0
	_, err = db.GetEngine(ctx).Select("SUM(file_compressed_size) AS size").
		Table("action_artifact").
		Where("owner_id = ?", userID).
		Get(&size)
	if err != nil {
		return 0, err
	}
	totalSize += size

	size = 0
	_, err = db.GetEngine(ctx).Select("SUM(package_blob.size) AS size").
		Table("package_blob").
		Join("INNER", "`package_file`", "`package_file`.blob_id = `package_blob`.id").
		Join("INNER", "`package_version`", "`package_file`.version_id = `package_version`.id").
		Join("INNER", "`package`", "`package_version`.package_id = `package`.id").
		Where("`package`.owner_id = ?", userID).
		Get(&size)
	if err != nil {
		return 0, err
	}
	totalSize += size

	return totalSize, nil
}
