// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package convert

import (
	"context"
	"strconv"

	issue_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	api "code.gitea.io/gitea/modules/structs"
)

func ToQuotaUsedAttachmentList(ctx context.Context, attachments []*repo_model.Attachment) (*api.QuotaUsedAttachmentList, error) {
	getAttachmentContainer := func(a *repo_model.Attachment) (string, string, error) {
		if a.ReleaseID != 0 {
			release, err := repo_model.GetReleaseByID(ctx, a.ReleaseID)
			if err != nil {
				return "", "", err
			}
			if err = release.LoadAttributes(ctx); err != nil {
				return "", "", err
			}
			return release.APIURL(), release.HTMLURL(), nil
		}
		if a.CommentID != 0 {
			comment, err := issue_model.GetCommentByID(ctx, a.CommentID)
			if err != nil {
				return "", "", err
			}
			return comment.APIURL(ctx), comment.HTMLURL(ctx), nil
		}
		if a.IssueID != 0 {
			issue, err := issue_model.GetIssueByID(ctx, a.IssueID)
			if err != nil {
				return "", "", err
			}
			if err = issue.LoadRepo(ctx); err != nil {
				return "", "", err
			}
			return issue.APIURL(ctx), issue.HTMLURL(), nil
		}
		return "", "", nil
	}

	result := make(api.QuotaUsedAttachmentList, len(attachments))
	for i, a := range attachments {
		capiURL, chtmlURL, err := getAttachmentContainer(a)
		if err != nil {
			return nil, err
		}

		apiURL := capiURL + "/assets/" + strconv.FormatInt(a.ID, 10)
		result[i] = &api.QuotaUsedAttachment{
			Name: a.Name,
			Size: a.Size,
			APIURL: apiURL,
		}
		result[i].ContainedIn.APIURL = capiURL
		result[i].ContainedIn.HTMLURL = chtmlURL
	}

	return &result, nil
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
