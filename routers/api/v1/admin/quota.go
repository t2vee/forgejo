// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

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
	// swagger:operation GET /admin/quota/groups admin adminListQuotaGroups
	// ---
	// summary: List the available quota groups
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaGroupList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	groups, err := quota_model.ListQuotaGroups(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListQuotaGroups", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToQuotaGroupList(ctx, groups))
}

// CreateQuotaGroup creates a new quota group
func CreateQuotaGroup(ctx *context.APIContext) {
	// swagger:operation POST /admin/quota/groups admin adminCreateQuotaGroup
	// ---
	// summary: Create a new quota group
	// produces:
	// - application/json
	// parameters:
	// - name: name
	//   in: body
	//   description: name of the quota group
	//   type: string
	//   required: true
	// - name: limit_git
	//   in: body
	//   description: Git quota limit for the group (in bytes)
	//   type: integer
	//   required: true
	// - name: limit_files
	//   in: body
	//   description: File asset quota limit for the group (in bytes)
	//   type: integer
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/created"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.CreateQuotaGroupOption)

	err := quota_model.CreateQuotaGroup(ctx, *form)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "CreateQuotaGroup", err)
		return
	}
	ctx.Status(http.StatusCreated)
}

// ListUsersInQuotaGroup lists all the users in a quota group
func ListUsersInQuotaGroup(ctx *context.APIContext) {
	// swagger:operation GET /admin/quota/groups/{quotagroup}/users admin adminListUsersInQuotaGroup
	// ---
	// summary: List users in a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to list members of
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/UserList"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	users, err := quota_model.ListUsersInQuotaGroup(ctx, ctx.QuotaGroup.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "ListUsersInQuotaGroup", err)
		return
	}
	ctx.JSON(http.StatusOK, convert.ToUsers(ctx, ctx.Doer, users))
}

// AddUserToQuotaGroup adds a user to a quota group
func AddUserToQuotaGroup(ctx *context.APIContext) {
	// swagger:operation POST /admin/quota/groups/{quotagroup}/users admin adminAddUserToQuotaGroup
	// ---
	// summary: Add a user to a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to add the user to
	//   type: string
	//   required: true
	// - name: username
	//   in: body
	//   description: username of the user to add to the quota group
	//   type: string
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/created"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

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

// RemoveUserFromQuotaGroup removes a user from a quota group
func RemoveUserFromQuotaGroup(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/quota/groups/{quotagroup}/users admin adminRemoveUserFromQuotaGroup
	// ---
	// summary: Remove a user from a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to remove a user from
	//   type: string
	//   required: true
	// - name: username
	//   in: body
	//   description: username of the user to remove from the quota group
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

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

// DeleteQuotaGroup deletes a quota group
func DeleteQuotaGroup(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/quota/groups/{quotagroup} admin adminDeleteQuotaGroup
	// ---
	// summary: Delete a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to delete
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

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

// GetQuotaGroup returns information about a quota group
func GetQuotaGroup(ctx *context.APIContext) {
	// swagger:operation GET /admin/quota/groups/{quotagroup} admin adminGetQuotaGroup
	// ---
	// summary: Get information about the quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to query
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaGroup"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

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
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"

	gitUse, err := quota_model.GetGitUseForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetGitUseForUser", err)
		return
	}
	fileUse, err := quota_model.GetFilesUseForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetFilesUseForUser", err)
		return
	}

	limits, err := quota_model.GetQuotaLimitsForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser", err)
	}

	groups, err := quota_model.GetQuotaGroupsForUser(ctx, ctx.ContextUser.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetUserQuota", err)
		return
	}

	userQuota := api.UserQuota{
		GitLimit:  limits.LimitGit,
		GitUse:    gitUse,
		FileLimit: limits.LimitFiles,
		FileUse:   fileUse,
	}
	if groups != nil {
		userQuota.Groups = make([]string, len(groups))
		for i, group := range groups {
			userQuota.Groups[i] = group.Name
		}
	}
	ctx.JSON(http.StatusOK, &userQuota)
}
