// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"code.gitea.io/gitea/models/db"
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

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
