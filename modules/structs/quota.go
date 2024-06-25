// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package structs

// QuotaGroupAddOrRemoveUserOption represents the options for quota group membership management
type QuotaGroupAddOrRemoveUserOption struct {
	// Name of the user to add to or remove from the quota group
	// required: true
	Username string `json:"username" binding:"Required"`
}
