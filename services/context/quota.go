// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package context

import (
	"net/http"
	"strings"

	quota_model "code.gitea.io/gitea/models/quota"
	"code.gitea.io/gitea/modules/base"
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

func (ctx *Context) QuotaExceeded() {
	showHTML := false
	for _, part := range ctx.Req.Header["Accept"] {
		if strings.Contains(part, "text/html") {
			showHTML = true
			break
		}
	}
	if !showHTML {
		ctx.plainTextInternal(3, http.StatusRequestEntityTooLarge, []byte("Quota exceeded.\n"))
		return
	}

	ctx.Data["IsRepo"] = ctx.Repo.Repository != nil
	ctx.Data["Title"] = "Quota Exceeded"
	ctx.HTML(http.StatusRequestEntityTooLarge, base.TplName("status/413"))
}

func EnforceGitQuotaWeb() func(ctx *Context) {
	return func(ctx *Context) {
		ok, err := quota_model.CheckGitQuotaLimitsForUser(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "CheckGitQuotaLimitsForUser")
			return
		}
		if !ok {
			ctx.QuotaExceeded()
			return
		}
	}
}

func EnforceFilesQuotaWeb() func(ctx *Context) {
	return func(ctx *Context) {
		ok, err := quota_model.CheckFilesQuotaLimitsForUser(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "CheckFilesQuotaLimitsForUser")
			return
		}
		if !ok {
			ctx.QuotaExceeded()
			return
		}
	}
}

func (ctx *APIContext) QuotaExceeded() {
	ctx.JSON(http.StatusRequestEntityTooLarge, APIError{
		Message: "quota exceeded",
		URL:     setting.API.SwaggerURL,
	})
}

func EnforceFilesQuotaAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		ok, err := quota_model.CheckFilesQuotaLimitsForUser(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.InternalServerError(err)
			return
		}
		if !ok {
			ctx.QuotaExceeded()
			return
		}
	}
}

func EnforceGitQuotaAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		ok, err := quota_model.CheckGitQuotaLimitsForUser(ctx, ctx.Doer.ID)
		if err != nil {
			ctx.InternalServerError(err)
			return
		}
		if !ok {
			ctx.QuotaExceeded()
			return
		}
	}
}
