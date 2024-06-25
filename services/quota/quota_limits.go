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
	QuotaLimitCategoryAssetAttachmentsReleases
	QuotaLimitCategoryAssetAttachmentsIssues
	QuotaLimitCategoryAssetArtifacts
	QuotaLimitCategoryAssetPackages
	QuotaLimitCategoryWiki
)

// QuotaLimits represents the limits affecting a user
// swagger:model
type QuotaLimits struct { //revive:disable-line:exported
	// The total space available to the user
	Total  *int64             `json:"total,omitempty"`
	Git    *QuotaLimitsGit    `json:"git,omitempty"`
	Assets *QuotaLimitsAssets `json:"assets,omitempty"`
}

// QuotaLimitsGit represents the Git-related limits affecting a user
// swagger:model
type QuotaLimitsGit struct { //revive:disable-line:exported
	// The total git space available to the user
	Total *int64 `json:"total,omitempty"`
	// Normal git space available to the user
	Code *int64 `json:"code,omitempty"`
	// Git LFS space available to the user
	LFS *int64 `json:"lfs,omitempty"`
}

// QuotaLimitsAssets represents the asset-related limits affecting a user
// swagger:model
type QuotaLimitsAssets struct { //revive:disable-line:exported
	// The total amount of asset space available to the user
	Total *int64 `json:"total,omitempty"`
	// Space available to the user for attachments
	Attachments *QuotaLimitsAttachments `json:"attachments,omitempty"`
	// Space available to the user for artifacts
	Artifacts *int64 `json:"artifacts,omitempty"`
	// Space available to the user for packages
	Packages *int64 `json:"packages,omitempty"`
}

// QuotaLimitsAttachments represents the attachment related limits affecting a user
// swagger:model
type QuotaLimitsAttachments struct { //revive:disable-line:exported
	// Total amount of attachment space available to the user
	Total *int64 `json:"total,omitempty"`
	// Space available to the user for release attachments
	Releases *int64 `json:"releases,omitempty"`
	// Space available to the user for issue & comment attachments
	Issues *int64 `json:"issues,omitempty"`
}

func (s *QuotaLimitsGit) IsEmpty() bool {
	return s.Total == nil && s.Code == nil && s.LFS == nil
}

func (s *QuotaLimitsAssets) IsEmpty() bool {
	return s.Total == nil && s.Attachments == nil && s.Artifacts == nil && s.Packages == nil
}

func (s *QuotaLimitsAttachments) IsEmpty() bool {
	return s.Total == nil && s.Releases == nil && s.Issues == nil
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
	pickTotal := func(outer, inner *int64) *int64 {
		if outer != nil {
			return outer
		}
		return inner
	}

	switch category {
	case QuotaLimitCategoryGitCode:
		return pick(l.Git.Total, l.Git.Code)
	case QuotaLimitCategoryGitLFS:
		return pick(l.Git.Total, l.Git.LFS)
	case QuotaLimitCategoryGitTotal:
		return pick(l.Git.Total, l.Git.Code, l.Git.LFS)

	case QuotaLimitCategoryAssetAttachmentsReleases:
		return pick(pickTotal(l.Assets.Total, l.Assets.Attachments.Total), l.Assets.Attachments.Releases)
	case QuotaLimitCategoryAssetAttachmentsIssues:
		return pick(pickTotal(l.Assets.Total, l.Assets.Attachments.Total), l.Assets.Attachments.Issues)
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
		Git: &QuotaLimitsGit{},
		Assets: &QuotaLimitsAssets{
			Attachments: &QuotaLimitsAttachments{},
		},
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
			limits.Assets.Attachments.Releases = maxOf(limits.Assets.Attachments.Releases, group.LimitAssetAttachmentsReleases)
			limits.Assets.Attachments.Issues = maxOf(limits.Assets.Attachments.Issues, group.LimitAssetAttachmentsIssues)
			limits.Assets.Packages = maxOf(limits.Assets.Packages, group.LimitAssetPackages)
			limits.Assets.Artifacts = maxOf(limits.Assets.Artifacts, group.LimitAssetArtifacts)
		}
	}

	if limits.Git.IsEmpty() {
		limits.Git = nil
	}
	if limits.Assets.Attachments.IsEmpty() {
		limits.Assets.Attachments = nil
	}
	if limits.Assets.IsEmpty() {
		limits.Assets = nil
	}

	return &limits, nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
