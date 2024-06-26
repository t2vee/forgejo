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

func TestQuotaLimits(t *testing.T) {
	var (
		gitCategories = []quota_service.QuotaLimitCategory{
			quota_service.QuotaLimitCategoryGitTotal,
			quota_service.QuotaLimitCategoryGitCode,
			quota_service.QuotaLimitCategoryGitLFS,
		}
		assetAttachmentCategories = []quota_service.QuotaLimitCategory{
			quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
			quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
			quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
		}
		assetCategories = append(
			assetAttachmentCategories,
			quota_service.QuotaLimitCategoryAssetTotal,
			quota_service.QuotaLimitCategoryAssetArtifacts,
			quota_service.QuotaLimitCategoryAssetPackages,
		)

		allCategories = append(
			gitCategories,
			append(assetCategories, assetAttachmentCategories...)...,
		)
	)

	assertCategoryAndLimit := func(t *testing.T, limits quota_service.QuotaLimits, limitCategory, expectedCategory quota_service.QuotaLimitCategory, expectedLimit int64) {
		t.Helper()

		limit, category := limits.GetLimitForCategory(limitCategory)
		assert.Equal(t, expectedCategory.String(), category.String())
		assert.EqualValues(t, expectedLimit, limit)
	}

	assertUniformCategoryAndLimit := func(t *testing.T, limits quota_service.QuotaLimits, categories []quota_service.QuotaLimitCategory, expectedCategory quota_service.QuotaLimitCategory, expectedLimit int64) {
		t.Helper()

		for _, c := range categories {
			assertCategoryAndLimit(t, limits, c, expectedCategory, expectedLimit)
		}
	}

	assertUniformLimitAndUnchangedCategory := func(t *testing.T, limits quota_service.QuotaLimits, categories []quota_service.QuotaLimitCategory, expectedLimit int64) {
		t.Helper()

		for _, c := range categories {
			assertCategoryAndLimit(t, limits, c, c, expectedLimit)
		}
	}

	t.Run("no groups", func(t *testing.T) {
		groups := []*quota_model.QuotaGroup{}
		limits := quota_service.GetQuotaLimitsForGroups(groups)

		assert.Nil(t, limits.Total)
		assert.Nil(t, limits.Git)
		assert.Nil(t, limits.Assets)

		assertUniformLimitAndUnchangedCategory(t, limits, allCategories, -1)
	})

	t.Run("single group", func(t *testing.T) {
		limitsForSingleGroup := func(group quota_model.QuotaGroup) quota_service.QuotaLimits {
			groups := []*quota_model.QuotaGroup{&group}
			return quota_service.GetQuotaLimitsForGroups(groups)
		}

		t.Run("total", func(t *testing.T) {
			total := int64(1024)
			limits := limitsForSingleGroup(quota_model.QuotaGroup{LimitTotal: &total})

			assert.NotNil(t, limits.Total)
			assert.Nil(t, limits.Git)
			assert.Nil(t, limits.Assets)

			assertUniformCategoryAndLimit(t, limits, allCategories, quota_service.QuotaLimitCategoryTotal, total)
		})

		t.Run("git", func(t *testing.T) {
			t.Run("total", func(t *testing.T) {
				total := int64(1024)
				limits := limitsForSingleGroup(quota_model.QuotaGroup{LimitGitTotal: &total})

				assert.NotNil(t, limits.Git)
				assert.NotNil(t, limits.Git.Total)
				assert.Nil(t, limits.Git.Code)
				assert.Nil(t, limits.Git.LFS)

				assertUniformCategoryAndLimit(t, limits, gitCategories, quota_service.QuotaLimitCategoryGitTotal, total)
				assertUniformLimitAndUnchangedCategory(t, limits, assetCategories, -1)
			})

			t.Run("code", func(t *testing.T) {
				value := int64(1024)
				limits := limitsForSingleGroup(quota_model.QuotaGroup{LimitGitCode: &value})

				assert.NotNil(t, limits.Git)
				assert.Nil(t, limits.Git.Total)
				assert.NotNil(t, limits.Git.Code)
				assert.Nil(t, limits.Git.LFS)


				assertCategoryAndLimit(t, limits, quota_service.QuotaLimitCategoryGitCode, quota_service.QuotaLimitCategoryGitCode, value)
				assertCategoryAndLimit(t, limits, quota_service.QuotaLimitCategoryGitLFS, quota_service.QuotaLimitCategoryGitLFS, -1)
			})
		})

	})
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
