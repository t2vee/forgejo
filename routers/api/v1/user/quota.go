// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package user

import (
	"net/http"

	quota_model "code.gitea.io/gitea/models/quota"
	"code.gitea.io/gitea/routers/api/v1/utils"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
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
	//     "$ref": "#/responses/QuotaInfo"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	used, err := quota_model.GetUsedForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.GetUsedForUser", err)
		return
	}

	groups, err := quota_model.GetGroupsForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.GetGroupsForUser", err)
		return
	}

	result := convert.ToQuotaInfo(used, groups)
	ctx.JSON(http.StatusOK, &result)
}

// CheckQuota returns whether the authenticated user is over the subject quota
func CheckQuota(ctx *context.APIContext) {
	// swagger:operation GET /user/quota/check user userCheckQuota
	// ---
	// summary: Check if the authenticated user is over quota for a given subject
	// produces:
	// - application/json
	// responses:
	//   "200":
	//     "$ref": "#/responses/boolean"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"
	subjectQuery := ctx.FormTrim("subject")

	subject, err := quota_model.ParseLimitSubject(subjectQuery)
	if err != nil {
		ctx.Error(http.StatusUnprocessableEntity, "quota_model.ParseLimitSubject", err)
		return
	}

	ok, err := quota_model.EvaluateForUser(ctx, ctx.Doer.ID, subject)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "quota_model.EvaluateForUser", err)
		return
	}

	ctx.JSON(http.StatusOK, &ok)
}

// ListQuotaAttachments lists attachments affecting the authenticated user's quota
func ListQuotaAttachments(ctx *context.APIContext) {
	// swagger:operation GET /user/quota/attachments user userListQuotaAttachments
	// ---
	// summary: List the attachments affecting the authenticated user's quota
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaUsedAttachmentList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	opts := utils.GetListOptions(ctx)
	count, attachments, err := quota_model.GetQuotaAttachmentsForUser(ctx, ctx.Doer.ID, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaAttachmentsForUser", err)
		return
	}

	result, err := convert.ToQuotaUsedAttachmentList(ctx, *attachments)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "convert.ToQuotaUsedAttachmentList", err)
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, result)
}

// ListQuotaPackages lists packages affecting the authenticated user's quota
func ListQuotaPackages(ctx *context.APIContext) {
	// swagger:operation GET /user/quota/packages user userListQuotaPackages
	// ---
	// summary: List the packages affecting the authenticated user's quota
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaUsedPackageList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	opts := utils.GetListOptions(ctx)
	count, packages, err := quota_model.GetQuotaPackagesForUser(ctx, ctx.Doer.ID, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaPackagesForUser", err)
		return
	}

	result, err := convert.ToQuotaUsedPackageList(ctx, *packages)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "convert.ToQuotaUsedPackageList", err)
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, result)
}

// ListQuotaArtifacts lists artifacts affecting the authenticated user's quota
func ListQuotaArtifacts(ctx *context.APIContext) {
	// swagger:operation GET /user/quota/artifacts user userListQuotaArtifacts
	// ---
	// summary: List the artifacts affecting the authenticated user's quota
	// produces:
	// - application/json
	// parameters:
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/QuotaUsedArtifactList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	opts := utils.GetListOptions(ctx)
	count, artifacts, err := quota_model.GetQuotaArtifactsForUser(ctx, ctx.Doer.ID, opts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaArtifactsForUser", err)
		return
	}

	result, err := convert.ToQuotaUsedArtifactList(ctx, *artifacts)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "convert.ToQuotaUsedArtifactList", err)
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, result)
}
