// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package user

import (
	"net/http"

	"code.gitea.io/gitea/services/context"
	quota_service "code.gitea.io/gitea/services/quota"
)

// GetQuota returns the quota information for the authenticated user
func GetQuota(ctx *context.APIContext) {
	// swagger:operation GET /user/quota user userGetQuota
	// ---
	// summary: Get quota information for the authenticated user
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserQuota"

	used, err := quota_service.GetQuotaUsedForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaUsedForUser", err)
		return
	}

	limits, err := quota_service.GetQuotaLimitsForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser", err)
		return
	}

	result := quota_service.UserQuota{
		Limits: *limits,
		Used:   *used,
	}

	ctx.JSON(http.StatusOK, &result)
}
