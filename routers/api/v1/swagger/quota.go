// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package swagger

import (
	quota_model "code.gitea.io/gitea/models/quota"
	api "code.gitea.io/gitea/modules/structs"
	quota_service "code.gitea.io/gitea/services/quota"
)

// UserQuota
// swagger:response UserQuota
type swaggerResponseUserQuota struct {
	// in:body
	Body quota_service.UserQuota `json:"body"`
}

// QuotaGroup
// swagger:response QuotaGroup
type swaggerResponseQuotaGroup struct {
	// in:body
	Body quota_model.QuotaGroup `json:"body"`
}

// QuotaGroupList
// swagger:response QuotaGroupList
type swaggerResponseQuotaGroupList struct {
	// in:body
	Body quota_model.QuotaGroupList `json:"body"`
}

// QuotaGroupAddOrRemoveUserOption
// swagger:response QuotaGroupAddOrRemoveUserOption
type swaggerDefinitionQuotaGroupAddOrRemoveUserOption struct {
	// in:body
	Body api.QuotaGroupAddOrRemoveUserOption `json:"body"`
}
