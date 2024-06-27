// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"

	"code.gitea.io/gitea/modules/setting"
)

func IsUsedWithinLimits(used *QuotaUsed, limits *QuotaLimits, category QuotaLimitCategory) bool {
	n, itemLimits, categories := limits.ResolveForCategory(category)
	for i := range n {
		if itemLimits[i] == -1 {
			continue
		}

		itemUsed := used.GetUsedForCategory(categories[i])
		if itemUsed > itemLimits[i] {
			return false
		}
	}

	return true
}

func IsWithinQuotaLimit(ctx context.Context, userID int64, category QuotaLimitCategory) (bool, error) {
	if !setting.Quota.Enabled {
		return true, nil
	}

	limits, err := GetQuotaLimitsForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	used, err := GetQuotaUsedForUser(ctx, userID)
	if err != nil {
		return false, err
	}

	return IsUsedWithinLimits(used, limits, category), nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
