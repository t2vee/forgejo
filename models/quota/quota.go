// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package quota

import (
	"context"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"

	"xorm.io/builder"
)

type QuotaKind int //revive:disable-line:exported

const (
	QuotaKindUser QuotaKind = iota
)

type QuotaLimitCategory int //revive:disable-line:exported

const (
	QuotaLimitCategoryGitTotal QuotaLimitCategory = iota
	QuotaLimitCategoryGitCode
	QuotaLimitCategoryGitLFS
	QuotaLimitCategoryAssetAttachments
	QuotaLimitCategoryAssetArtifacts
	QuotaLimitCategoryAssetPackages
	QuotaLimitCategoryWiki
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

type QuotaMapping struct { //revive:disable-line:exported
	ID           int64 `xorm:"pk autoincr"`
	Kind         QuotaKind
	MappedID     int64
	QuotaGroupID int64
}

type QuotaLimits struct { //revive:disable-line:exported
	LimitTotal *int64

	LimitGitTotal *int64
	LimitGitCode  *int64
	LimitGitLFS   *int64

	LimitAssetTotal       *int64
	LimitAssetAttachments *int64
	LimitAssetPackages    *int64
	LimitAssetArtifacts   *int64
}

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

func (l *QuotaLimits) getLimitForCategory(category QuotaLimitCategory) int64 {
	pick := func(specificTotal *int64, specifics ...*int64) int64 {
		if l.LimitTotal != nil {
			return *l.LimitTotal
		}
		if specificTotal != nil {
			return *specificTotal
		}

		var (
			sum   int64
			found bool
		)

		for _, num := range specifics {
			if num != nil {
				sum += *num
				found = true
			}
		}
		if !found {
			return -1
		}
		return sum
	}

	switch category {
	case QuotaLimitCategoryGitCode:
		return pick(l.LimitGitTotal, l.LimitGitCode)
	case QuotaLimitCategoryGitLFS:
		return pick(l.LimitGitTotal, l.LimitGitLFS)
	case QuotaLimitCategoryGitTotal:
		return pick(l.LimitGitTotal, l.LimitGitCode, l.LimitGitLFS)

	case QuotaLimitCategoryAssetAttachments:
		return pick(l.LimitAssetTotal, l.LimitAssetAttachments)
	case QuotaLimitCategoryAssetArtifacts:
		return pick(l.LimitAssetTotal, l.LimitAssetArtifacts)
	case QuotaLimitCategoryAssetPackages:
		return pick(l.LimitAssetTotal, l.LimitAssetPackages)

	case QuotaLimitCategoryWiki:
		return pick(nil, nil)
	}

	return pick(nil, nil)
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
		var group QuotaGroup
		has, err := db.GetEngine(ctx).Where("name = ?", setting.Quota.DefaultGroup).Get(&group)
		if err != nil {
			return nil, err
		}
		if !has {
			return nil, nil
		}
		groups = []*QuotaGroup{&group}
	}

	return groups, nil
}

func GetQuotaLimitsForUser(ctx context.Context, userID int64) (*QuotaLimits, error) {
	groups, err := GetQuotaGroupsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	limits := QuotaLimits{}
	if len(groups) > 0 {
		var minusOne int64 = -1
		maxOf := func(old, new *int64) *int64 {
			if old == nil && new == nil {
				return nil
			}
			if old == nil && new != nil {
				return new
			}
			if old != nil && new == nil {
				return old
			}

			if *new == -1 || *old == -1 {
				return &minusOne
			}

			if *new > *old {
				return new
			} else {
				return old
			}
		}

		for _, group := range groups {
			limits.LimitGitTotal = maxOf(limits.LimitGitTotal, group.LimitGitTotal)
			limits.LimitGitCode = maxOf(limits.LimitGitCode, group.LimitGitCode)
			limits.LimitGitLFS = maxOf(limits.LimitGitLFS, group.LimitGitLFS)

			limits.LimitAssetTotal = maxOf(limits.LimitAssetTotal, group.LimitAssetTotal)
			limits.LimitAssetAttachments = maxOf(limits.LimitAssetAttachments, group.LimitAssetAttachments)
			limits.LimitAssetPackages = maxOf(limits.LimitAssetPackages, group.LimitAssetPackages)
			limits.LimitAssetArtifacts = maxOf(limits.LimitAssetArtifacts, group.LimitAssetArtifacts)
		}
	}

	return &limits, nil
}

func IsWithinQuotaLimit(ctx context.Context, userID int64, category QuotaLimitCategory) (bool, error) {
	if !setting.Quota.Enabled {
		return true, nil
	}

	limits, err := GetQuotaLimitsForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	used, err := GetQuotaUsedForUser(ctx, userID)
	if err != nil {
		return false, err
	}

	// Determine the comparison participants
	itemLimit := limits.getLimitForCategory(category)
	if itemLimit == -1 {
		return true, nil
	}
	if itemLimit == 0 {
		return false, nil
	}

	itemUsed := used.getUsedForCategory(category)

	return itemUsed < itemLimit, nil
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

// UserQuota represents a user's quota info
// swagger:model
type UserQuota struct {
	Limits QuotaLimits `json:"limits"`
	Used   QuotaUsed   `json:"used"`
	Groups []string    `json:"groups,omitempty"`
}

// QuotaGroupList is a list of quota groups
// swagger:model
type QuotaGroupList []*QuotaGroup
