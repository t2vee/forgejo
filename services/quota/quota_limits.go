// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"

	quota_model "code.gitea.io/gitea/models/quota"
)

type QuotaLimitCategory int //revive:disable-line:exported

const (
	QuotaLimitCategoryTotal QuotaLimitCategory = iota
	QuotaLimitCategoryGitTotal
	QuotaLimitCategoryGitCode
	QuotaLimitCategoryGitLFS
	QuotaLimitCategoryAssetTotal
	QuotaLimitCategoryAssetAttachmentsTotal
	QuotaLimitCategoryAssetAttachmentsReleases
	QuotaLimitCategoryAssetAttachmentsIssues
	QuotaLimitCategoryAssetArtifacts
	QuotaLimitCategoryAssetPackages
	QuotaLimitCategoryWiki
)

func (l QuotaLimitCategory) String() string {
	switch l {
	case QuotaLimitCategoryTotal:
		return "total"
	case QuotaLimitCategoryGitTotal:
		return "git-total"
	case QuotaLimitCategoryGitCode:
		return "git-code"
	case QuotaLimitCategoryGitLFS:
		return "git-lfs"
	case QuotaLimitCategoryAssetTotal:
		return "asset-total"
	case QuotaLimitCategoryAssetAttachmentsTotal:
		return "asset-attachments-total"
	case QuotaLimitCategoryAssetAttachmentsReleases:
		return "asset-attachments-release"
	case QuotaLimitCategoryAssetAttachmentsIssues:
		return "asset-attachments-issues"
	case QuotaLimitCategoryAssetArtifacts:
		return "asset-artifacts"
	case QuotaLimitCategoryAssetPackages:
		return "asset-packages"
	case QuotaLimitCategoryWiki:
		return "wiki"
	}
	return "<unknown>"
}

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

func (l *QuotaLimits) getLimitForCategory(category QuotaLimitCategory) (int64, QuotaLimitCategory) {
	pick := func(specificCategoryTotal QuotaLimitCategory, specificTotal *int64, specifics ...*int64) (int64, QuotaLimitCategory) {
		if l.Total != nil {
			return *l.Total, QuotaLimitCategoryTotal
		}
		if specificTotal != nil {
			return *specificTotal, specificCategoryTotal
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
			return -1, category
		}
		return sum, category
	}
	pickTotal := func(outer, inner *int64, outerCategory, innerCategory QuotaLimitCategory) (*int64, QuotaLimitCategory) {
		if outer != nil {
			return outer, outerCategory
		}
		return inner, innerCategory
	}

	switch category {
	case QuotaLimitCategoryGitCode:
		return pick(QuotaLimitCategoryGitTotal, l.Git.GetTotal(), l.Git.GetCode())
	case QuotaLimitCategoryGitLFS:
		return pick(QuotaLimitCategoryGitTotal, l.Git.GetTotal(), l.Git.GetLFS())
	case QuotaLimitCategoryGitTotal:
		return pick(QuotaLimitCategoryGitTotal, l.Git.GetTotal(), l.Git.GetCode(), l.Git.GetLFS())

	case QuotaLimitCategoryAssetAttachmentsReleases:
		total, category := pickTotal(
			l.Assets.GetTotal(), l.Assets.GetAttachments().GetTotal(),
			QuotaLimitCategoryAssetTotal, QuotaLimitCategoryAssetAttachmentsTotal,
		)
		return pick(category, total, l.Assets.GetAttachments().GetReleases())
	case QuotaLimitCategoryAssetAttachmentsIssues:
		total, category := pickTotal(
			l.Assets.GetTotal(), l.Assets.GetAttachments().GetTotal(),
			QuotaLimitCategoryAssetTotal, QuotaLimitCategoryAssetAttachmentsTotal,
		)
		return pick(category, total, l.Assets.GetAttachments().GetIssues())
	case QuotaLimitCategoryAssetArtifacts:
		return pick(QuotaLimitCategoryAssetTotal, l.Assets.GetTotal(), l.Assets.GetArtifacts())
	case QuotaLimitCategoryAssetPackages:
		return pick(QuotaLimitCategoryAssetTotal, l.Assets.GetTotal(), l.Assets.GetPackages())

	case QuotaLimitCategoryWiki:
		return pick(category, nil, nil)
	}

	return pick(category, nil, nil)
}

func GetQuotaLimitsForUser(ctx context.Context, userID int64) (*QuotaLimits, error) {
	groups, err := GetQuotaGroupsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	limits := getQuotaLimitsForGroups(groups)
	return &limits, nil
}

func getQuotaLimitsForGroups(groups []*quota_model.QuotaGroup) QuotaLimits {
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

			limits.Git.Total = maxOf(limits.Git.GetTotal(), group.LimitGitTotal)
			limits.Git.Code = maxOf(limits.Git.GetCode(), group.LimitGitCode)
			limits.Git.LFS = maxOf(limits.Git.GetLFS(), group.LimitGitLFS)

			limits.Assets.Total = maxOf(limits.Assets.GetTotal(), group.LimitAssetTotal)
			limits.Assets.Attachments.Releases = maxOf(limits.Assets.GetAttachments().GetReleases(), group.LimitAssetAttachmentsReleases)
			limits.Assets.Attachments.Issues = maxOf(limits.Assets.GetAttachments().GetIssues(), group.LimitAssetAttachmentsIssues)
			limits.Assets.Packages = maxOf(limits.Assets.GetPackages(), group.LimitAssetPackages)
			limits.Assets.Artifacts = maxOf(limits.Assets.GetArtifacts(), group.LimitAssetArtifacts)
		}
	}

	if limits.Git.IsEmpty() {
		limits.Git = nil
	}
	if limits.Assets.GetAttachments().IsEmpty() {
		limits.Assets.Attachments = nil
	}
	if limits.Assets.IsEmpty() {
		limits.Assets = nil
	}

	return limits
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

func (l *QuotaLimitsGit) GetTotal() *int64 {
	if l == nil {
		return nil
	}
	return l.Total
}

func (l *QuotaLimitsGit) GetCode() *int64 {
	if l == nil {
		return nil
	}
	return l.Code
}

func (l *QuotaLimitsGit) GetLFS() *int64 {
	if l == nil {
		return nil
	}
	return l.LFS
}

func (l *QuotaLimitsAssets) GetTotal() *int64 {
	if l == nil {
		return nil
	}
	return l.Total
}

func (l *QuotaLimitsAssets) GetArtifacts() *int64 {
	if l == nil {
		return nil
	}
	return l.Artifacts
}

func (l *QuotaLimitsAssets) GetAttachments() *QuotaLimitsAttachments {
	if l == nil {
		return nil
	}
	return l.Attachments
}

func (l *QuotaLimitsAssets) GetPackages() *int64 {
	if l == nil {
		return nil
	}
	return l.Packages
}

func (l *QuotaLimitsAttachments) GetTotal() *int64 {
	if l == nil {
		return nil
	}
	return l.Total
}

func (l *QuotaLimitsAttachments) GetReleases() *int64 {
	if l == nil {
		return nil
	}
	return l.Releases
}

func (l *QuotaLimitsAttachments) GetIssues() *int64 {
	if l == nil {
		return nil
	}
	return l.Issues
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
