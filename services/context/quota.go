// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package context

import (
	"net/http"
	"strings"

	quota_model "code.gitea.io/gitea/models/quota"
	"code.gitea.io/gitea/modules/base"
)

type QuotaTargetType int

const (
	QuotaTargetUser QuotaTargetType = iota
	QuotaTargetRepo
)

// QuotaGroupAssignmentAPI returns a middleware to handle context-quota-group assignment for api routes
func QuotaGroupAssignmentAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		groupName := ctx.Params("quotagroup")
		group, err := quota_model.GetGroupByName(ctx, groupName)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "quota_model.GetGroupByName", err)
			return
		}
		if group == nil {
			ctx.NotFound()
			return
		}
		ctx.QuotaGroup = group
	}
}

func QuotaRuleAssignmentAPI() func(ctx *APIContext) {
	return func(ctx *APIContext) {
		ruleName := ctx.Params("quotarule")
		rule, err := quota_model.GetRuleByName(ctx, ruleName)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "quota_model.GetRuleByName", err)
			return
		}
		if rule == nil {
			ctx.NotFound()
			return
		}
		ctx.QuotaRule = rule
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

func EnforceQuotaWeb(subject quota_model.LimitSubject) func(ctx *Context) {
	return func(ctx *Context) {
		ok, err := quota_model.EvaluateForUser(ctx, ctx.Doer.ID, subject)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "quota_model.EvaluateForUser")
			return
		}
		if !ok {
			ctx.QuotaExceeded()
		}
	}
}

// QuotaExceeded
// swagger:response quotaExceeded
type APIQuotaExceeded struct {
	Message string `json:"message"`
}

func (ctx *APIContext) QuotaExceeded() {
	ctx.JSON(http.StatusRequestEntityTooLarge, APIQuotaExceeded{
		Message: "quota exceeded",
	})
}

func EnforceQuotaAPI(subject quota_model.LimitSubject, target QuotaTargetType) func(ctx *APIContext) {
	return func(ctx *APIContext) {
		var userID int64
		switch target {
		case QuotaTargetUser:
			userID = ctx.Doer.ID
		case QuotaTargetRepo:
			userID = ctx.Repo.Owner.ID
		}
		ok, err := quota_model.EvaluateForUser(ctx, userID, subject)
		if err != nil {
			ctx.InternalServerError(err)
			return
		}
		if !ok {
			ctx.QuotaExceeded()
		}
	}
}
