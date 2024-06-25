// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"
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

type QuotaLimits struct { //revive:disable-line:exported
	Total  *int64             `json:"total,omitempty"`
	Git    *QuotaLimitsGit    `json:"git,omitempty"`
	Assets *QuotaLimitsAssets `json:"assets,omitempty"`
}

type QuotaLimitsGit struct {
	Total *int64 `json:"total,omitempty"`
	Code  *int64 `json:"code,omitempty"`
	LFS   *int64 `json:"lfs,omitempty"`
}

type QuotaLimitsAssets struct {
	Total       *int64 `json:"total,omitempty"`
	Attachments *int64 `json:"attachments,omitempty"`
	Artifacts   *int64 `json:"artifacts,omitempty"`
	Packages    *int64 `json:"packages,omitempty"`
}

func (s *QuotaLimitsGit) IsEmpty() bool {
	return s.Total == nil && s.Code == nil && s.LFS == nil
}

func (s *QuotaLimitsAssets) IsEmpty() bool {
	return s.Total == nil && s.Attachments == nil && s.Artifacts == nil && s.Packages == nil
}

func (l *QuotaLimits) getLimitForCategory(category QuotaLimitCategory) int64 {
	pick := func(specificTotal *int64, specifics ...*int64) int64 {
		if l.Total != nil {
			return *l.Total
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
		return pick(l.Git.Total, l.Git.Code)
	case QuotaLimitCategoryGitLFS:
		return pick(l.Git.Total, l.Git.LFS)
	case QuotaLimitCategoryGitTotal:
		return pick(l.Git.Total, l.Git.Code, l.Git.LFS)

	case QuotaLimitCategoryAssetAttachments:
		return pick(l.Assets.Total, l.Assets.Attachments)
	case QuotaLimitCategoryAssetArtifacts:
		return pick(l.Assets.Total, l.Assets.Artifacts)
	case QuotaLimitCategoryAssetPackages:
		return pick(l.Assets.Total, l.Assets.Packages)

	case QuotaLimitCategoryWiki:
		return pick(nil, nil)
	}

	return pick(nil, nil)
}

func GetQuotaLimitsForUser(ctx context.Context, userID int64) (*QuotaLimits, error) {
	groups, err := GetQuotaGroupsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	limits := QuotaLimits{
		Git:    &QuotaLimitsGit{},
		Assets: &QuotaLimitsAssets{},
	}
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
			}
			return old
		}

		for _, group := range groups {
			limits.Total = maxOf(limits.Total, group.LimitTotal)

			limits.Git.Total = maxOf(limits.Git.Total, group.LimitGitTotal)
			limits.Git.Code = maxOf(limits.Git.Code, group.LimitGitCode)
			limits.Git.LFS = maxOf(limits.Git.LFS, group.LimitGitLFS)

			limits.Assets.Total = maxOf(limits.Assets.Total, group.LimitAssetTotal)
			limits.Assets.Attachments = maxOf(limits.Assets.Attachments, group.LimitAssetAttachments)
			limits.Assets.Packages = maxOf(limits.Assets.Packages, group.LimitAssetPackages)
			limits.Assets.Artifacts = maxOf(limits.Assets.Artifacts, group.LimitAssetArtifacts)
		}
	}

	if limits.Git.IsEmpty() {
		limits.Git = nil
	}
	if limits.Assets.IsEmpty() {
		limits.Assets = nil
	}

	return &limits, nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
