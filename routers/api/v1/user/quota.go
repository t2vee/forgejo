// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package user

import (
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
	"code.gitea.io/gitea/services/context"
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

	used, err := quota_model.GetQuotaUsedForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetGitUseForUser", err)
		return
	}

	limits, err := quota_model.GetQuotaLimitsForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser", err)
		return
	}

	result := quota_model.UserQuota{
		Limits: *limits,
		Used:   *used,
	}

	ctx.JSON(http.StatusOK, &result)
}
