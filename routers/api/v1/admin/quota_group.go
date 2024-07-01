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

	groups, err := quota_model.ListGroups(ctx)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.ListGroups", err)
		return
	}
	for _, group := range groups {
		if err = group.LoadRules(ctx); err != nil {
			ctx.Error(http.StatusInternalServerError, "quota_model.group.LoadRules", err)
			return
		}
	}

	ctx.JSON(http.StatusOK, convert.ToQuotaGroupList(groups))
}

// CreateQuotaGroup creates a new quota group
func CreateQuotaGroup(ctx *context.APIContext) {
	// swagger:operation POST /admin/quota/groups admin adminCreateQuotaGroup
	// ---
	// summary: Create a new quota group
	// produces:
	// - application/json
	// parameters:
	// - name: group
	//   in: body
	//   description: Definition of the quota group
	//   schema:
	//     "$ref": "#/definitions/CreateQuotaGroupOptions"
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"

	form := web.GetForm(ctx).(*api.CreateQuotaGroupOptions)

	err := quota_model.CreateGroup(ctx, form.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.CreateGroup", err)
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

	users, err := quota_model.ListUsersInGroup(ctx, ctx.QuotaGroup.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.ListUsersInGroup", err)
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
	//   schema:
	//     "$ref": "#/definitions/QuotaGroupAddOrRemoveUserOption"
	//   required: true
	// responses:
	//   "201":
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

	err = ctx.QuotaGroup.AddUserByID(ctx, user.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_group.group.AddUserByID", err)
		return
	}
	ctx.Status(http.StatusCreated)
}

// RemoveUserFromQuotaGroup removes a user from a quota group
func RemoveUserFromQuotaGroup(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/quota/groups/{quotagroup}/users/{username} admin adminRemoveUserFromQuotaGroup
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
	//   in: path
	//   description: username of the user to add to the quota group
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

	username := ctx.Params("username")
	if username == "" {
		ctx.NotFound()
		return
	}

	user, err := user_model.GetUserByName(ctx, username)
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
		ctx.Error(http.StatusInternalServerError, "quota_model.group.RemoveUserByID", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// SetUserQuotaGroups moves the user to specific quota groups
func SetUserQuotaGroups(ctx *context.APIContext) {
	// swagger:operation POST /admin/users/{username}/quota/groups admin adminSetUserQuotaGroups
	// ---
	// summary: Set the user's quota groups to a given list.
	// produces:
	// - application/json
	// parameters:
	// - name: username
	//   in: path
	//   description: username of the user to add to the quota group
	//   type: string
	//   required: true
	// - name: groups
	//   in: body
	//   description: quota group to remove a user from
	//   schema:
	//     "$ref": "#/definitions/SetUserQuotaGroupsOptions"
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

	form := web.GetForm(ctx).(*api.SetUserQuotaGroupsOptions)

	err := quota_model.SetUserGroups(ctx, ctx.ContextUser.ID, form.Groups)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.SetUserGroups", err)
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

	err := quota_model.DeleteGroupByName(ctx, ctx.QuotaGroup.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.DeleteGroupByName", err)
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

	ctx.JSON(http.StatusOK, convert.ToQuotaGroup(*ctx.QuotaGroup))
}

// AddRuleToQuotaGroup adds a rule to a quota group
func AddRuleToQuotaGroup(ctx *context.APIContext) {
	// swagger:operation POST /admin/quota/groups/{quotagroup}/rules admin adminAddRuleToQuotaGroup
	// ---
	// summary: Adds a rule to a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to add a rule to
	//   type: string
	//   required: true
	// - name: name
	//   in: body
	//   description: the name of the quota rule to add to the group
	//   schema:
	//     "$ref": "#/definitions/AddRuleToQuotaGroupOptions"
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	form := web.GetForm(ctx).(*api.AddRuleToQuotaGroupOptions)

	err := ctx.QuotaGroup.AddRuleByName(ctx, form.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.group.AddFormByName", err)
		return
	}
	ctx.Status(http.StatusCreated)
}

// RemoveRuleFromQuotaGroup removes a rule from a quota group
func RemoveRuleFromQuotaGroup(ctx *context.APIContext) {
	// swagger:operation DELETE /admin/quota/groups/{quotagroup}/rules/{quotarule} admin adminRemoveRuleFromQuotaGroup
	// ---
	// summary: Removes a rule from a quota group
	// produces:
	// - application/json
	// parameters:
	// - name: quotagroup
	//   in: path
	//   description: quota group to add a rule to
	//   type: string
	//   required: true
	// - name: quotarule
	//   in: path
	//   description: the name of the quota rule to remove from the group
	//   type: string
	//   required: true
	// responses:
	//   "201":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	err := ctx.QuotaGroup.RemoveRuleByName(ctx, ctx.QuotaRule.Name)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.group.RemoveRuleByName", err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
