// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota_test

import (
	"testing"

	quota_model "code.gitea.io/gitea/models/quota"
	quota_service "code.gitea.io/gitea/services/quota"

	"github.com/stretchr/testify/assert"
)

func makeTestUsed() quota_service.QuotaUsed {
	used := quota_service.QuotaUsed{}
	used.Git.Code = 10240
	used.Git.LFS = 102400
	used.Assets.Attachments.Issues = 256
	used.Assets.Attachments.Releases = 2048
	used.Assets.Artifacts = 512
	used.Assets.Packages = 4096

	return used
}

func assertQuotaCheck(t *testing.T, used quota_service.QuotaUsed, limits quota_service.QuotaLimits, passingCategories, failingCategories []quota_service.QuotaLimitCategory) {
	t.Helper()

	for _, category := range passingCategories {
		t.Run("passing:"+category.String(), func(t *testing.T) {
			result := quota_service.IsUsedWithinLimits(&used, &limits, category)
			assert.True(t, result)
		})
	}

	for _, category := range failingCategories {
		t.Run("failing:"+category.String(), func(t *testing.T) {
			result := quota_service.IsUsedWithinLimits(&used, &limits, category)
			assert.False(t, result)
		})
	}
}

func TestQuotaUsedWithoutLimits(t *testing.T) {
	used := makeTestUsed()
	groups := []*quota_model.QuotaGroup{}
	limits := quota_service.GetQuotaLimitsForGroups(groups)

	assertQuotaCheck(t, used, limits,
		[]quota_service.QuotaLimitCategory{
			quota_service.QuotaLimitCategoryTotal,
			quota_service.QuotaLimitCategoryGitTotal,
			quota_service.QuotaLimitCategoryGitCode,
			quota_service.QuotaLimitCategoryGitLFS,
			quota_service.QuotaLimitCategoryAssetTotal,
			quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
			quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
			quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
			quota_service.QuotaLimitCategoryAssetArtifacts,
			quota_service.QuotaLimitCategoryAssetPackages,
			quota_service.QuotaLimitCategoryWiki,
		},
		[]quota_service.QuotaLimitCategory{},
	)
}

func TestQuotaUsedWithTotal0(t *testing.T) {
	used := makeTestUsed()
	groups := []*quota_model.QuotaGroup{
		{
			LimitTotal: Ptr(int64(0)),
		},
	}
	limits := quota_service.GetQuotaLimitsForGroups(groups)

	assertQuotaCheck(t, used, limits,
		[]quota_service.QuotaLimitCategory{},
		[]quota_service.QuotaLimitCategory{
			quota_service.QuotaLimitCategoryTotal,
			quota_service.QuotaLimitCategoryGitTotal,
			quota_service.QuotaLimitCategoryGitCode,
			quota_service.QuotaLimitCategoryGitLFS,
			quota_service.QuotaLimitCategoryAssetTotal,
			quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
			quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
			quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
			quota_service.QuotaLimitCategoryAssetArtifacts,
			quota_service.QuotaLimitCategoryAssetPackages,
			quota_service.QuotaLimitCategoryWiki,
		},
	)
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
