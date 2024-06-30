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

func init() {
	db.RegisterModel(new(Rule))
	db.RegisterModel(new(Group))
	db.RegisterModel(new(GroupRuleMapping))
	db.RegisterModel(new(GroupMapping))
}

func EvaluateForUser(ctx context.Context, userID int64, subject LimitSubject) (bool, error) {
	if !setting.Quota.Enabled {
		return true, nil
	}

	groups, err := GetGroupsForUser(ctx, userID)
	if err != nil {
		return false, err
	}

	used, err := GetUsedForUser(ctx, userID)
	if err != nil {
		return false, err
	}

	return groups.Evaluate(*used, subject), nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
