// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"

	quota_model "code.gitea.io/gitea/models/quota"
	api "code.gitea.io/gitea/modules/structs"
)

// ToQuotaGroup converts quota_model.QuotaGroup to api.QuotaGroup
func ToQuotaGroup(ctx context.Context, quotaGroup *quota_model.QuotaGroup) *api.QuotaGroup {
	if quotaGroup == nil {
		return nil
	}

	return &api.QuotaGroup{
		Name:       quotaGroup.Name,
		LimitGit:   quotaGroup.LimitGit,
		LimitFiles: quotaGroup.LimitFiles,
	}
}

// ToQuotaGroupList converts a list of quota_model.QuotaGroup to api.QuotaGroupList
func ToQuotaGroupList(ctx context.Context, quotaGroups []*quota_model.QuotaGroup) api.QuotaGroupList {
	result := make([]*api.QuotaGroup, len(quotaGroups))
	for i := range quotaGroups {
		result[i] = ToQuotaGroup(ctx, quotaGroups[i])
	}
	return result
}
