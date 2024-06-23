// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// import (
// 	"time"

// 	"code.gitea.io/gitea/modules/json"
// )

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

type CreateQuotaGroupOption struct {
	Name       string `json:"name" binding:"Required"`
	LimitGit   int64  `json:"limit_git" binding:"Required"`
	LimitFiles int64  `json:"limit_files" binding:"Required"`
}

type QuotaGroup struct {
	Name       string `json:"name"`
	LimitGit   int64  `json:"limit_git"`
	LimitFiles int64  `json:"limit_files"`
}

type QuotaGroupList []*QuotaGroup

type QuotaGroupAddOrRemoveUserOption struct {
	Username string `json:"username" binding:"Required"`
}
