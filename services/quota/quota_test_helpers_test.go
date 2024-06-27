// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota_test

import (
	quota_service "code.gitea.io/gitea/services/quota"
)

func Ptr[T any](v T) *T {
	return &v
}

func makeCatList(start, end quota_service.QuotaLimitCategory) []quota_service.QuotaLimitCategory {
	list := make([]quota_service.QuotaLimitCategory, end-start+1)
	for i := start; i <= end; i++ {
		list[i-start] = i
	}
	return list
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
