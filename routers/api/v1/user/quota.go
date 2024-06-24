// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
	api "code.gitea.io/gitea/modules/structs"
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

	gitUse, err := quota_model.GetGitUseForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetGitUseForUser", err)
		return
	}
	fileUse, err := quota_model.GetFilesUseForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetFilesUseForUser", err)
		return
	}

	limits, err := quota_model.GetQuotaLimitsForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser", err)
		return
	}

	userQuota := api.UserQuota{
		GitLimit:  limits.LimitGit,
		GitUse:    gitUse,
		FileLimit: limits.LimitFiles,
		FileUse:   fileUse,
	}
	ctx.JSON(http.StatusOK, &userQuota)
}
