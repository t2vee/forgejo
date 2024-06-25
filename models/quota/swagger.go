// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

// UserQuota represents a user's quota info
// swagger:model
type UserQuota struct {
	Limits QuotaLimits `json:"limits"`
	Used   QuotaUsed   `json:"used"`
	// quota groups the user is part of
	Groups []string `json:"groups,omitempty"`
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
