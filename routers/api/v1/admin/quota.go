// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/services/context"
)

// GetUserQuota return information about a user's quota
func GetUserQuota(ctx *context.APIContext) {
	// swagger:operation GET /admin/users/{username}/quota admin adminGetUserQuota
	// ---
	// summary: Get the user's quota info
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of user to query
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserQuota"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	gitUse, err := quota_model.GetGitUseForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetUserQuota", err)
		return
	}
	fileUse, err := quota_model.GetFilesUseForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetUserQuota", err)
		return
	}

	userQuota := api.UserQuota{
		GitUse: gitUse,
		FileUse: fileUse,
	}
	ctx.JSON(http.StatusOK, &userQuota)
}
