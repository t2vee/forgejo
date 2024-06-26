// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota_test

import (
	"maps"
	"testing"

	quota_model "code.gitea.io/gitea/models/quota"
	quota_service "code.gitea.io/gitea/services/quota"

	"github.com/stretchr/testify/assert"
)

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

	allCategories = append([]quota_service.QuotaLimitCategory{
		quota_service.QuotaLimitCategoryTotal,
	},
		append(
			gitCategories,
			append(assetCategories, assetAttachmentCategories...)...,
		)...,
	)
)

func Ptr[T any](v T) *T {
	return &v
}

type TestExpectation struct {
	N          int64
	Limits     []int64
	Categories []quota_service.QuotaLimitCategory
}
type (
	TestExpectations map[quota_service.QuotaLimitCategory]TestExpectation
	TestCase         struct {
		Group    quota_model.QuotaGroup
		Expected TestExpectations
	}
)

func repeatExpectations(expectation TestExpectation, categories ...quota_service.QuotaLimitCategory) TestExpectations {
	expectations := make(TestExpectations, quota_service.QuotaLimitCategoryEnd+1)

	for _, category := range categories {
		expectations[category] = expectation
	}
	return expectations
}

func makeExpectationForCategory(category quota_service.QuotaLimitCategory, expectation TestExpectation) TestExpectations {
	expectations := make(TestExpectations, quota_service.QuotaLimitCategoryEnd+1)

	expectations[category] = expectation
	return expectations
}

func makeCatList(start, end quota_service.QuotaLimitCategory) []quota_service.QuotaLimitCategory {
	list := make([]quota_service.QuotaLimitCategory, end-start+1)
	for i := start; i <= end; i++ {
		list[i-start] = i
	}
	return list
}

func mergeExpectations(expectations ...TestExpectations) TestExpectations {
	result := TestExpectations{}
	for _, es := range expectations {
		maps.Copy(result, es)
	}
	return result
}

func assertCategories(t *testing.T, expectedCategories, actualCategories []quota_service.QuotaLimitCategory) {
	t.Helper()

	for i := range len(expectedCategories) {
		assert.Equal(t, expectedCategories[i].String(), actualCategories[i].String())
	}
}

func runTestCases(t *testing.T, testCases map[string]TestCase) {
	t.Helper()

	limitsForSingleGroup := func(group quota_model.QuotaGroup) quota_service.QuotaLimits {
		groups := []*quota_model.QuotaGroup{&group}
		return quota_service.GetQuotaLimitsForGroups(groups)
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			limits := limitsForSingleGroup(testCase.Group)

			for category, expectation := range testCase.Expected {
				n, itemLimits, itemCategories := limits.ResolveForCategory(category)
				assert.EqualValues(t, expectation.N, n)
				assert.EqualValues(t, expectation.Limits, itemLimits)
				assertCategories(t, expectation.Categories, itemCategories)
			}
		})
	}
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
