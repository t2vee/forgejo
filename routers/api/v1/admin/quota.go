// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/web"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
)

// ListQuotaGroups returns all the quota groups
func ListQuotaGroups(ctx *context.APIContext) {
	groups, err := quota_model.ListQuotaGroups(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListQuotaGroups", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToQuotaGroupList(ctx, groups))
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
	users, err := quota_model.ListUsersInQuotaGroup(ctx, ctx.QuotaGroup.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListUsersInQuotaGroup", err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToUsers(ctx, ctx.Doer, users))
}

func AddUserToQuotaGroup(ctx *context.APIContext) {
	form := web.GetForm(ctx).(*api.QuotaGroupAddOrRemoveUserOption)

	user, err := user_model.GetUserByName(ctx, form.Username)
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.NotFound("GetUserByName", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetUserByName", err)
		}
		return
	}

	err = ctx.QuotaGroup.AddUserByID(ctx, user.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "AddUserToQuotaGroup", err)
		return
	}
	ctx.Status(http.StatusCreated)
}

func RemoveUserFromQuotaGroup(ctx *context.APIContext) {
	form := web.GetForm(ctx).(*api.QuotaGroupAddOrRemoveUserOption)

	user, err := user_model.GetUserByName(ctx, form.Username)
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.NotFound("GetUserByName", err)
		} else {
			ctx.Error(http.StatusInternalServerError, "GetUserByName", err)
		}
		return
	}

	err = ctx.QuotaGroup.RemoveUserByID(ctx, user.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "RemoveUserFromQuotaGroup", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func DeleteQuotaGroup(ctx *context.APIContext) {
	if quota_model.IsQuotaGroupInUse(ctx, ctx.QuotaGroup.Name) {
		ctx.Error(http.StatusUnprocessableEntity, "DeleteQuotaGroup", "cannot delete quota group that is in use")
		return
	}

	err := quota_model.DeleteQuotaGroupByName(ctx, ctx.QuotaGroup.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "DeleteQuotaGroup", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func GetQuotaGroup(ctx *context.APIContext) {
	ctx.JSON(http.StatusOK, convert.ToQuotaGroup(ctx, ctx.QuotaGroup))
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
