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
	t.Run("no groups", func(t *testing.T) {
		groups := []*quota_model.QuotaGroup{}
		limits := quota_service.GetQuotaLimitsForGroups(groups)

		assert.Nil(t, limits.Total)
		assert.Nil(t, limits.Git)
		assert.Nil(t, limits.Assets)

		for _, category := range allCategories {
			n, _, _ := limits.ResolveForCategory(category)
			assert.EqualValues(t, 0, n)
		}
	})

	t.Run("single group", func(t *testing.T) {
		t.Run("single limit", func(t *testing.T) {
			tests := map[string]TestCase{
				"Total": {
					Group: quota_model.QuotaGroup{LimitTotal: Ptr(int64(1024))},
					// Expectation: Every category is checked against Total
					Expected: repeatExpectations(
						TestExpectation{
							N:      1,
							Limits: []int64{1024},
							Categories: []quota_service.QuotaLimitCategory{
								quota_service.QuotaLimitCategoryTotal,
							},
						},
						makeCatList(quota_service.QuotaLimitCategoryStart, quota_service.QuotaLimitCategoryEnd)...,
					),
				},
				"GitTotal": {
					Group: quota_model.QuotaGroup{LimitGitTotal: Ptr(int64(1024))},
					// Expectation: Every Git category (& Total) is checked
					// against LimitGitTotal. The rest aren't checked.
					Expected: mergeExpectations(
						repeatExpectations(
							TestExpectation{
								N:      1,
								Limits: []int64{1024},
								Categories: []quota_service.QuotaLimitCategory{
									quota_service.QuotaLimitCategoryGitTotal,
								},
							},
							makeCatList(quota_service.QuotaLimitCategoryStart, quota_service.QuotaLimitCategoryGitLFS)...,
						),
						repeatExpectations(
							TestExpectation{},
							makeCatList(quota_service.QuotaLimitCategoryAssetTotal, quota_service.QuotaLimitCategoryEnd)...,
						),
					),
				},
				"GitCode": {
					Group: quota_model.QuotaGroup{LimitGitCode: Ptr(int64(1024))},
					// Expectation: Total, GitTotal, and GitCode are checked
					// against GitCode. The rest aren't checked.
					Expected: mergeExpectations(
						repeatExpectations(
							TestExpectation{
								N:      1,
								Limits: []int64{1024},
								Categories: []quota_service.QuotaLimitCategory{
									quota_service.QuotaLimitCategoryGitCode,
								},
							},
							makeCatList(quota_service.QuotaLimitCategoryStart, quota_service.QuotaLimitCategoryGitCode)...,
						),
						repeatExpectations(
							TestExpectation{},
							makeCatList(quota_service.QuotaLimitCategoryGitLFS, quota_service.QuotaLimitCategoryEnd)...,
						),
					),
				},
				"GitLFS": {
					Group: quota_model.QuotaGroup{LimitGitLFS: Ptr(int64(1024))},
					// Expectation: Total, GitTotal, and GitLFS are checked
					// against GitLFS. The rest aren't checked.
					Expected: mergeExpectations(
						repeatExpectations(
							TestExpectation{
								N:      1,
								Limits: []int64{1024},
								Categories: []quota_service.QuotaLimitCategory{
									quota_service.QuotaLimitCategoryGitLFS,
								},
							},
							quota_service.QuotaLimitCategoryTotal,
							quota_service.QuotaLimitCategoryGitTotal,
							quota_service.QuotaLimitCategoryGitLFS,
						),
						makeExpectationForCategory(
							quota_service.QuotaLimitCategoryGitCode,
							TestExpectation{},
						),
						repeatExpectations(
							TestExpectation{},
							makeCatList(quota_service.QuotaLimitCategoryAssetTotal, quota_service.QuotaLimitCategoryEnd)...,
						),
					),
				},
			}

			runTestCases(t, tests)
		})
	})
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
