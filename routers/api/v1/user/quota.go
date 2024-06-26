// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package user

import (
	"net/http"

	"code.gitea.io/gitea/routers/api/v1/utils"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/convert"
	quota_service "code.gitea.io/gitea/services/quota"
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

	used, err := quota_service.GetQuotaUsedForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaUsedForUser", err)
		return
	}

	limits, err := quota_service.GetQuotaLimitsForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "GetQuotaLimitsForUser", err)
		return
	}

	result := quota_service.UserQuota{
		Limits: *limits,
		Used:   *used,
	}

	ctx.JSON(http.StatusOK, &result)
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

	opts := utils.GetListOptions(ctx)
	count, attachments, err := quota_service.GetQuotaAttachmentsForUser(ctx, ctx.Doer.ID, opts)
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

	opts := utils.GetListOptions(ctx)
	count, packages, err := quota_service.GetQuotaPackagesForUser(ctx, ctx.Doer.ID, opts)
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
