// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/setting"
)

type QuotaKind int //revive:disable-line:exported

const (
	QuotaKindUser QuotaKind = iota
)

type QuotaMapping struct { //revive:disable-line:exported
	ID           int64 `xorm:"pk autoincr"`
	Kind         QuotaKind
	MappedID     int64
	QuotaGroupID int64
}

func init() {
	db.RegisterModel(new(QuotaGroup))
	db.RegisterModel(new(QuotaMapping))
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

	// Determine the comparison participants
	itemLimit := limits.getLimitForCategory(category)
	if itemLimit == -1 {
		return true, nil
	}
	if itemLimit == 0 {
		return false, nil
	}

	itemUsed := used.getUsedForCategory(category)

	return itemUsed < itemLimit, nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
