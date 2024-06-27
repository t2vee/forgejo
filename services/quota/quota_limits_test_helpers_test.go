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
	"github.com/stretchr/testify/require"
)

type LimitTestExpectation struct {
	N          int64
	Limits     []int64
	Categories []quota_service.QuotaLimitCategory
}
type (
	LimitTestExpectations map[quota_service.QuotaLimitCategory]LimitTestExpectation
	LimitTestCase         struct {
		Groups   []*quota_model.QuotaGroup
		Expected LimitTestExpectations
	}
)

func repeatLimitExpectations(expectation LimitTestExpectation, categories ...quota_service.QuotaLimitCategory) LimitTestExpectations {
	expectations := make(LimitTestExpectations, quota_service.QuotaLimitCategoryEnd+1)

	for _, category := range categories {
		expectations[category] = expectation
	}
	return expectations
}

func makeLimitExpectationForCategory(category quota_service.QuotaLimitCategory, expectation LimitTestExpectation) LimitTestExpectations {
	expectations := make(LimitTestExpectations, quota_service.QuotaLimitCategoryEnd+1)

	expectations[category] = expectation
	return expectations
}

func mergeLimitExpectations(expectations ...LimitTestExpectations) LimitTestExpectations {
	result := LimitTestExpectations{}
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

func runLimitTestCases(t *testing.T, testCases map[string]LimitTestCase) {
	t.Helper()

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			limits := quota_service.GetQuotaLimitsForGroups(testCase.Groups)

			for category := range quota_service.QuotaLimitCategoryEnd {
				t.Run("resolve-for:"+category.String(), func(t *testing.T) {
					expectation, ok := testCase.Expected[category]
					if !ok {
						expectation = LimitTestExpectation{}
					}

					n, itemLimits, itemCategories := limits.ResolveForCategory(category)

					require.EqualValues(t, expectation.N, n, "n != expectation.N")
					assert.EqualValues(t, expectation.Limits, itemLimits)
					assertCategories(t, expectation.Categories, itemCategories)
				})
			}
		})
	}
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
