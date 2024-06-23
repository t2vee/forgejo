// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "code.gitea.io/gitea/modules/structs"
)

// UserQuota
// swagger:response UserQuota
type swaggerResponseUserQuota struct {
	// in:body
	Body api.UserQuota `json:"body"`
}
