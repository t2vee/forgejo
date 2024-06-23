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
	// Git storage used by the user (including LFS)
	GitUse int64 `json:"git_use"`
	// File storage used by the user (attachments, artifacts, and packages)
	FileUse int64 `json:"file_use"`
}
