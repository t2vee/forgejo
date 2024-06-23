// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/context"
)

// ListQuotaGroups returns all the quota groups
func ListQuotaGroups(ctx *context.APIContext) {
	groups, err := quota_model.ListQuotaGroups(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListQuotaGroups", err)
		return
	}

	ctx.JSON(http.StatusOK, groups)
}

func CreateQuotaGroup(ctx *context.APIContext) {
	form := web.GetForm(ctx).(*api.CreateQuotaGroupOption)

	err := quota_model.CreateQuotaGroup(ctx, *form)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateQuotaGroup", err)
		return
	}
	ctx.Status(http.StatusCreated)
}

func ListUsersInQuotaGroup(ctx *context.APIContext) {
	group := ctx.Params("name")

	users, err := quota_model.ListUsersInQuotaGroup(ctx, group)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListUsersInQuotaGroup", err)
		return
	}
	ctx.JSON(http.StatusOK, users)
}

func AddUserToQuotaGroup(ctx *context.APIContext) {
	group := ctx.Params("name")
	//username := ctx.Params("username")

	err := quota_model.AddUserToQuotaGroup(ctx, group, 1)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "AddUserToQuotaGroup", err)
		return
	}
	ctx.Status(http.StatusCreated)
}

func DeleteQuotaGroup(ctx *context.APIContext) {
	name := ctx.Params("name")

	if quota_model.IsQuotaGroupInUse(ctx, name) {
		ctx.Error(http.StatusUnprocessableEntity, "DeleteQuotaGroup", "cannot delete quota group that is in use")
		return
	}

	err := quota_model.DeleteQuotaGroupByName(ctx, name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteQuotaGroup", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func GetQuotaGroup(ctx *context.APIContext) {
	name := ctx.Params("name")

	group, err := quota_model.GetQuotaGroupByName(ctx, name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaGroup", err)
		return
	}

	if group == nil {
		ctx.Error(http.StatusNotFound, "GetQuotaGroup", "quota group not found")
		return
	}
	ctx.JSON(http.StatusOK, group)
}

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

	limits, err := quota_model.GetQuotaGroupForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetUserQuota", err)
		return
	}

	var userQuota api.UserQuota
	if limits != nil {
		userQuota = api.UserQuota{
			GitLimit: limits.LimitGit,
			GitUse: gitUse,
			FileLimit: limits.LimitFiles,
			FileUse: fileUse,
			Group: limits.Name,
		}
	} else {
		userQuota = api.UserQuota{
			GitUse: gitUse,
			FileUse: fileUse,
		}
	}
	ctx.JSON(http.StatusOK, &userQuota)
}
