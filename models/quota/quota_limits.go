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
	LimitTotal *int64

	LimitGitTotal *int64
	LimitGitCode  *int64
	LimitGitLFS   *int64

	LimitAssetTotal       *int64
	LimitAssetAttachments *int64
	LimitAssetPackages    *int64
	LimitAssetArtifacts   *int64
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
			}
			return old
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

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
