// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
	"code.gitea.io/gitea/modules/setting"
)

// QuotaGroupAssignmentAPI returns a middleware to handle context-quota-group assignment for api routes
func QuotaGroupAssignmentAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		groupName := ctx.Params("quotagroup")
		group, err := quota_model.GetQuotaGroupByName(ctx, groupName)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetQuotaGroupByName", err)
			return
		}
		if group == nil {
			ctx.NotFound()
			return
		}
		ctx.QuotaGroup = group
	}
}

// func (ctx *Context) EnforceFilesQuota() {

// }

func EnforceGitQuotaWeb() func(ctx *Context) {
	return func(ctx *Context) {
		if !setting.Quota.Enabled {
			return
		}

		limits, err := quota_model.GetQuotaLimitsForUser(ctx, ctx.Doer.ID)
		if err != nil {
			//log.Error("GetQuotaLimitsForUser: %v", err)
			ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser")
			return
		}

		if limits.LimitGit == -1 {
			return
		}
		if limits.LimitGit == 0 {
			ctx.QuotaExceeded()
			return
		}
		gitUse, err := quota_model.GetGitUseForUser(ctx, ctx.Doer.ID)
		if err != nil {
			//log.Error("GetFilesUseForUser: %v", err)
			ctx.Error(http.StatusInternalServerError, "GetGitUseForUser")
			return
		}
		if limits.LimitGit < gitUse {
			ctx.QuotaExceeded()
			return
		}
	}
}

func EnforceFilesQuotaWeb() func(ctx *Context) {
	return func(ctx *Context) {
		if !setting.Quota.Enabled {
			return
		}

		limits, err := quota_model.GetQuotaLimitsForUser(ctx, ctx.Doer.ID)
		if err != nil {
			//log.Error("GetQuotaLimitsForUser: %v", err)
			ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser")
			return
		}

		if limits.LimitFiles == -1 {
			return
		}
		if limits.LimitFiles == 0 {
			ctx.QuotaExceeded()
			return
		}
		filesUse, err := quota_model.GetFilesUseForUser(ctx, ctx.Doer.ID)
		if err != nil {
			//log.Error("GetFilesUseForUser: %v", err)
			ctx.Error(http.StatusInternalServerError, "GetFilesUseForUser")
			return
		}
		if limits.LimitFiles < filesUse {
			ctx.QuotaExceeded()
			return
		}
	}
}

func EnforceFilesQuotaAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		if !setting.Quota.Enabled {
			return
		}

		limits, err := quota_model.GetQuotaLimitsForUser(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser", err)
			return
		}

		if limits.LimitFiles == -1 {
			return
		}
		if limits.LimitFiles == 0 {
			ctx.JSON(http.StatusRequestEntityTooLarge, APIError{
				Message: "quota exceeded",
				URL: setting.API.SwaggerURL,
			})
			return
		}
		filesUse, err := quota_model.GetFilesUseForUser(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetFilesUseForUser", err)
			return
		}
		if limits.LimitFiles < filesUse {
				ctx.JSON(http.StatusRequestEntityTooLarge, APIError{
				Message: "quota exceeded",
				URL: setting.API.SwaggerURL,
			})
			return
		}
	}
}

func EnforceGitQuotaAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		if !setting.Quota.Enabled {
			return
		}

		limits, err := quota_model.GetQuotaLimitsForUser(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser", err)
			return
		}

		if limits.LimitGit == -1 {
			return
		}
		if limits.LimitGit == 0 {
			ctx.JSON(http.StatusRequestEntityTooLarge, APIError{
				Message: "quota exceeded",
				URL: setting.API.SwaggerURL,
			})
			return
		}
		gitUse, err := quota_model.GetGitUseForUser(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetGitUseForUser", err)
			return
		}
		if limits.LimitGit < gitUse {
				ctx.JSON(http.StatusRequestEntityTooLarge, APIError{
				Message: "quota exceeded",
				URL: setting.API.SwaggerURL,
			})
			return
		}
	}
}
