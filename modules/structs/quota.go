// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// UserQuota represents a user's quota info
// swagger:model
type UserQuota struct {
	// Git storage limit for the user (including LFS)
	GitLimit int64 `json:"git_limit"`
	// Git storage used by the user (including LFS)
	GitUse int64 `json:"git_use"`
	// File storage limit for the user
	FileLimit int64 `json:"file_limit"`
	// File storage used by the user (attachments, artifacts, and packages)
	FileUse int64 `json:"file_use"`
	// Quota group for the user
	Group string `json:"group,omitempty"`
}

// CreateQuotaGroupOption represents the options for creating a quota group
// swagger:model
type CreateQuotaGroupOption struct {
	// Name of the quota group
	Name string `json:"name" binding:"Required"`
	// Git storage limit for the group
	LimitGit int64 `json:"limit_git" binding:"Required"`
	// File storage limit for the group
	LimitFiles int64 `json:"limit_files" binding:"Required"`
}

// QuotaGroup represents a quota group
// swagger:model
type QuotaGroup struct {
	// Name of the quota group
	Name string `json:"name"`
	// Git storage limit for the group
	LimitGit int64 `json:"limit_git"`
	// File storage limit for the group
	LimitFiles int64 `json:"limit_files"`
}

// QuotaGroupList is a list of quota groups
// swagger:model
type QuotaGroupList []*QuotaGroup

// QuotaGroupAddOrRemoveUserOption represents the options for quota group membership management
// swagger:model
type QuotaGroupAddOrRemoveUserOption struct {
	// Name of the user to add to or remove from the quota group
	Username string `json:"username" binding:"Required"`
}
