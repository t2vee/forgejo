// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota_test

import (
	"testing"

	quota_model "code.gitea.io/gitea/models/quota"
	quota_service "code.gitea.io/gitea/services/quota"
)

func TestQuotaLimitsWithoutGroups(t *testing.T) {
	tests := map[string]LimitTestCase{
		"no groups": {
			Groups: []*quota_model.QuotaGroup{},
		},
	}

	runLimitTestCases(t, tests)
}

func g(group quota_model.QuotaGroup) []*quota_model.QuotaGroup {
	groups := []*quota_model.QuotaGroup{&group}
	return groups
}

func TestQuotaLimitsSingleGroupSingleLimit(t *testing.T) {
	tests := map[string]LimitTestCase{
		"Total": {
			Groups: g(quota_model.QuotaGroup{LimitTotal: Ptr(int64(1024))}),
			// Expectation: Every category is checked against Total
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
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
			Groups: g(quota_model.QuotaGroup{LimitGitTotal: Ptr(int64(1024))}),
			// Expectation: Every Git category (& Total) is checked
			// against LimitGitTotal. The rest aren't checked.
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{1024},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryGitTotal,
					},
				},
				makeCatList(quota_service.QuotaLimitCategoryStart, quota_service.QuotaLimitCategoryGitLFS)...,
			),
		},
		"GitCode": {
			Groups: g(quota_model.QuotaGroup{LimitGitCode: Ptr(int64(1024))}),
			// Expectation: Total, GitTotal, and GitCode are checked
			// against GitCode. The rest aren't checked.
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{1024},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryGitCode,
					},
				},
				makeCatList(quota_service.QuotaLimitCategoryStart, quota_service.QuotaLimitCategoryGitCode)...,
			),
		},
		"GitLFS": {
			Groups: g(quota_model.QuotaGroup{LimitGitLFS: Ptr(int64(1024))}),
			// Expectation: Total, GitTotal, and GitLFS are checked
			// against GitLFS. The rest aren't checked.
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
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
		},
		"AssetTotal": {
			Groups: g(quota_model.QuotaGroup{LimitAssetTotal: Ptr(int64(1024))}),
			// Expectation: Total, AssetTotal, and the rest of the Asset
			// category is checked against AssetTotal. The rest aren't
			// checked.
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{1024},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryAssetTotal,
					},
				},
				quota_service.QuotaLimitCategoryTotal,
				quota_service.QuotaLimitCategoryAssetTotal,
				quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
				quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
				quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
				quota_service.QuotaLimitCategoryAssetArtifacts,
				quota_service.QuotaLimitCategoryAssetPackages,
			),
		},
		"AssetAttachmentsTotal": {
			Groups: g(quota_model.QuotaGroup{LimitAssetAttachmentsTotal: Ptr(int64(1024))}),
			// Expectation: Total, AssetTotal, AssetAttachments* are
			// checked against AssetAttachmentsTotal. The rest aren't
			// checked.
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{1024},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
					},
				},
				quota_service.QuotaLimitCategoryTotal,
				quota_service.QuotaLimitCategoryAssetTotal,
				quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
				quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
				quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
			),
		},
		"AssetAttachmentsReleases": {
			Groups: g(quota_model.QuotaGroup{LimitAssetAttachmentsReleases: Ptr(int64(1024))}),
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{1024},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
					},
				},
				quota_service.QuotaLimitCategoryTotal,
				quota_service.QuotaLimitCategoryAssetTotal,
				quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
				quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
			),
		},
		"AssetAttachmentsIssues": {
			Groups: g(quota_model.QuotaGroup{LimitAssetAttachmentsIssues: Ptr(int64(1024))}),
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{1024},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
					},
				},
				quota_service.QuotaLimitCategoryTotal,
				quota_service.QuotaLimitCategoryAssetTotal,
				quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
				quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
			),
		},
		"AssetPackages": {
			Groups: g(quota_model.QuotaGroup{LimitAssetPackages: Ptr(int64(1024))}),
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{1024},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryAssetPackages,
					},
				},
				quota_service.QuotaLimitCategoryTotal,
				quota_service.QuotaLimitCategoryAssetTotal,
				quota_service.QuotaLimitCategoryAssetPackages,
			),
		},
		"AssetArtifacts": {
			Groups: g(quota_model.QuotaGroup{LimitAssetArtifacts: Ptr(int64(1024))}),
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{1024},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryAssetArtifacts,
					},
				},
				quota_service.QuotaLimitCategoryTotal,
				quota_service.QuotaLimitCategoryAssetTotal,
				quota_service.QuotaLimitCategoryAssetArtifacts,
			),
		},
	}

	runLimitTestCases(t, tests)
}

func TestQuotaSingleGroupComplexLimits(t *testing.T) {
	tests := map[string]LimitTestCase{
		"GitTotal + GitCode": {
			Groups: g(quota_model.QuotaGroup{
				LimitGitTotal: Ptr(int64(1024)),
				LimitGitCode:  Ptr(int64(2048)),
			}),
			Expected: mergeLimitExpectations(
				repeatLimitExpectations(
					LimitTestExpectation{
						N:      2,
						Limits: []int64{1024, 2048},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitTotal,
							quota_service.QuotaLimitCategoryGitCode,
						},
					},
					quota_service.QuotaLimitCategoryTotal,
					quota_service.QuotaLimitCategoryGitTotal,
					quota_service.QuotaLimitCategoryGitCode,
				),
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryGitLFS,
					LimitTestExpectation{
						N:      1,
						Limits: []int64{1024},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitTotal,
						},
					},
				),
			),
		},
		"AssetTotal + AttachmentsIssues": {
			Groups: g(quota_model.QuotaGroup{
				LimitAssetTotal:             Ptr(int64(1024)),
				LimitAssetAttachmentsIssues: Ptr(int64(2048)),
			}),
			Expected: mergeLimitExpectations(
				repeatLimitExpectations(
					LimitTestExpectation{
						N:      2,
						Limits: []int64{1024, 2048},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryAssetTotal,
							quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
						},
					},
					quota_service.QuotaLimitCategoryTotal,
					quota_service.QuotaLimitCategoryAssetTotal,
					quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
					quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
				),
				repeatLimitExpectations(
					LimitTestExpectation{
						N:      1,
						Limits: []int64{1024},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryAssetTotal,
						},
					},
					quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
					quota_service.QuotaLimitCategoryAssetPackages,
					quota_service.QuotaLimitCategoryAssetArtifacts,
				),
			),
		},
		"GitCode + GitLFS": {
			Groups: g(quota_model.QuotaGroup{
				LimitGitCode: Ptr(int64(1024)),
				LimitGitLFS:  Ptr(int64(2048)),
			}),
			Expected: mergeLimitExpectations(
				repeatLimitExpectations(
					LimitTestExpectation{
						N:      2,
						Limits: []int64{1024, 2048},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitCode,
							quota_service.QuotaLimitCategoryGitLFS,
						},
					},
					quota_service.QuotaLimitCategoryTotal,
					quota_service.QuotaLimitCategoryGitTotal,
				),
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryGitCode,
					LimitTestExpectation{
						N:      1,
						Limits: []int64{1024},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitCode,
						},
					},
				),
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryGitLFS,
					LimitTestExpectation{
						N:      1,
						Limits: []int64{2048},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitLFS,
						},
					},
				),
			),
		},
	}

	runLimitTestCases(t, tests)
}

func TestQuotaMultiGroupLimits(t *testing.T) {
	tests := map[string]LimitTestCase{
		"Disabled, Unlimited": {
			Groups: []*quota_model.QuotaGroup{
				{
					LimitTotal: Ptr(int64(0)),
				},
				{
					LimitTotal: Ptr(int64(-1)),
				},
			},
			Expected: repeatLimitExpectations(
				LimitTestExpectation{
					N:      1,
					Limits: []int64{-1},
					Categories: []quota_service.QuotaLimitCategory{
						quota_service.QuotaLimitCategoryTotal,
					},
				},
				makeCatList(quota_service.QuotaLimitCategoryStart, quota_service.QuotaLimitCategoryEnd)...,
			),
		},
		"Git*, AssetAttachments, AssetPackages": {
			Groups: []*quota_model.QuotaGroup{
				{
					LimitGitCode: Ptr(int64(1024)),
					LimitGitLFS:  Ptr(int64(2048)),
				},
				{
					LimitAssetAttachmentsTotal: Ptr(int64(512)),
				},
				{
					LimitAssetPackages: Ptr(int64(4096)),
				},
			},
			Expected: mergeLimitExpectations(
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryTotal,
					LimitTestExpectation{
						N:      4,
						Limits: []int64{1024, 2048, 512, 4096},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitCode,
							quota_service.QuotaLimitCategoryGitLFS,
							quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
							quota_service.QuotaLimitCategoryAssetPackages,
						},
					},
				),
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryGitTotal,
					LimitTestExpectation{
						N:      2,
						Limits: []int64{1024, 2048},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitCode,
							quota_service.QuotaLimitCategoryGitLFS,
						},
					},
				),
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryGitCode,
					LimitTestExpectation{
						N:      1,
						Limits: []int64{1024},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitCode,
						},
					},
				),
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryGitLFS,
					LimitTestExpectation{
						N:      1,
						Limits: []int64{2048},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryGitLFS,
						},
					},
				),
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryAssetTotal,
					LimitTestExpectation{
						N:      2,
						Limits: []int64{512, 4096},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
							quota_service.QuotaLimitCategoryAssetPackages,
						},
					},
				),
				repeatLimitExpectations(
					LimitTestExpectation{
						N:      1,
						Limits: []int64{512},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
						},
					},
					quota_service.QuotaLimitCategoryAssetAttachmentsTotal,
					quota_service.QuotaLimitCategoryAssetAttachmentsIssues,
					quota_service.QuotaLimitCategoryAssetAttachmentsReleases,
				),
				makeLimitExpectationForCategory(
					quota_service.QuotaLimitCategoryAssetPackages,
					LimitTestExpectation{
						N:      1,
						Limits: []int64{4096},
						Categories: []quota_service.QuotaLimitCategory{
							quota_service.QuotaLimitCategoryAssetPackages,
						},
					},
				),
			),
		},
	}

	runLimitTestCases(t, tests)
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
