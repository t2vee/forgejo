// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"context"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/foreignreference"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/modules/structs"
)

// InsertMilestones creates milestones of repository.
func InsertMilestones(ms ...*issues_model.Milestone) (err error) {
	if len(ms) == 0 {
		return nil
	}

	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)

	// to return the id, so we should not use batch insert
	for _, m := range ms {
		if _, err = sess.NoAutoTime().Insert(m); err != nil {
			return err
		}
	}

	if _, err = db.Exec(ctx, "UPDATE `repository` SET num_milestones = num_milestones + ? WHERE id = ?", len(ms), ms[0].RepoID); err != nil {
		return err
	}
	return committer.Commit()
}

// InsertIssues insert issues to database
func InsertIssues(issues ...*issues_model.Issue) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()

	for _, issue := range issues {
		if err := insertIssue(ctx, issue); err != nil {
			return err
		}
	}
	return committer.Commit()
}

func resolveIssueLabels(issueID int64, labels []*issues_model.Label) []issues_model.IssueLabel {
	issueLabels := make([]issues_model.IssueLabel, 0, len(labels))
	for _, label := range labels {
		issueLabels = append(issueLabels, issues_model.IssueLabel{
			IssueID: issueID,
			LabelID: label.ID,
		})
	}
	return issueLabels
}

func insertIssue(ctx context.Context, issue *issues_model.Issue) error {
	sess := db.GetEngine(ctx)
	if _, err := sess.NoAutoTime().Insert(issue); err != nil {
		return err
	}
	issueLabels := resolveIssueLabels(issue.ID, issue.Labels)
	if len(issueLabels) > 0 {
		if _, err := sess.Insert(issueLabels); err != nil {
			return err
		}
	}

	for _, reaction := range issue.Reactions {
		reaction.IssueID = issue.ID
	}

	if len(issue.Reactions) > 0 {
		if _, err := sess.Insert(issue.Reactions); err != nil {
			return err
		}
	}

	if issue.ForeignReference != nil {
		issue.ForeignReference.LocalIndex = issue.Index
		if _, err := sess.Insert(issue.ForeignReference); err != nil {
			return err
		}
	}

	return nil
}

// UpsertIssues creates new issues and updates existing issues in database
func UpsertIssues(issues ...*issues_model.Issue) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()

	for _, issue := range issues {
		if _, err := upsertIssue(ctx, issue); err != nil {
			return err
		}
	}
	return committer.Commit()
}

func updateIssue(ctx context.Context, issue *issues_model.Issue) error {
	sess := db.GetEngine(ctx)
	if _, err := sess.NoAutoTime().ID(issue.ID).Update(issue); err != nil {
		return err
	}
	issueLabels := resolveIssueLabels(issue.ID, issue.Labels)
	if len(issueLabels) > 0 {
		// delete old labels
		if _, err := sess.Where("issue_id = ?", issue.ID).Delete(); err != nil {
			return err
		}
		// insert new labels
		if _, err := sess.Insert(issueLabels); err != nil {
			return err
		}
	}

	for _, reaction := range issue.Reactions {
		reaction.IssueID = issue.ID
	}

	if len(issue.Reactions) > 0 {
		// update existing reactions and insert new ones
		for _, reaction := range issue.Reactions {
			exists, err := sess.Exist(&issues_model.Reaction{ID: reaction.ID})
			if err != nil {
				return err
			}
			if exists {
				if _, err := sess.ID(reaction.ID).Update(&reaction); err != nil {
					return err
				}
			} else {
				if _, err := sess.Insert(&reaction); err != nil {
					return err
				}
			}
		}
	}

	if issue.ForeignReference != nil {
		issue.ForeignReference.LocalIndex = issue.Index

		exists, err := sess.Exist(&foreignreference.ForeignReference{
			RepoID:     issue.ForeignReference.RepoID,
			LocalIndex: issue.ForeignReference.LocalIndex,
		})
		if err != nil {
			return err
		}

		if !exists {
			if _, err := sess.Insert(issue.ForeignReference); err != nil {
				return err
			}
		}
	}

	return nil
}

func upsertIssue(ctx context.Context, issue *issues_model.Issue) (isInsert bool, err error) {
	sess := db.GetEngine(ctx)

	exists, err := sess.Exist(&issues_model.Issue{ID: issue.ID})
	if err != nil {
		return false, err
	}

	if !exists {
		return true, insertIssue(ctx, issue)
	}
	return false, updateIssue(ctx, issue)
}

// InsertIssueComments inserts many comments of issues.
func InsertIssueComments(comments []*issues_model.Comment) error {
	if len(comments) == 0 {
		return nil
	}

	issueIDs := make(map[int64]bool)
	for _, comment := range comments {
		issueIDs[comment.IssueID] = true
	}

	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()
	for _, comment := range comments {
		if _, err := db.GetEngine(ctx).NoAutoTime().Insert(comment); err != nil {
			return err
		}

		for _, reaction := range comment.Reactions {
			reaction.IssueID = comment.IssueID
			reaction.CommentID = comment.ID
		}
		if len(comment.Reactions) > 0 {
			if err := db.Insert(ctx, comment.Reactions); err != nil {
				return err
			}
		}
	}

	for issueID := range issueIDs {
		if _, err := db.Exec(ctx, "UPDATE issue set num_comments = (SELECT count(*) FROM comment WHERE issue_id = ? AND `type`=?) WHERE id = ?",
			issueID, issues_model.CommentTypeComment, issueID); err != nil {
			return err
		}
	}
	return committer.Commit()
}

// InsertPullRequests inserted pull requests
func InsertPullRequests(prs ...*issues_model.PullRequest) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)
	for _, pr := range prs {
		if err := insertIssue(ctx, pr.Issue); err != nil {
			return err
		}
		pr.IssueID = pr.Issue.ID
		if _, err := sess.NoAutoTime().Insert(pr); err != nil {
			return err
		}
	}
	return committer.Commit()
}

// UpsertPullRequests inserts new pull requests and updates existing pull requests in database
func UpsertPullRequests(prs ...*issues_model.PullRequest) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)
	for _, pr := range prs {
		isInsert, err := upsertIssue(ctx, pr.Issue)
		if err != nil {
			return err
		}
		pr.IssueID = pr.Issue.ID

		if isInsert {
			if _, err := sess.NoAutoTime().Insert(pr); err != nil {
				return err
			}
		} else {
			if _, err := sess.NoAutoTime().ID(pr.ID).Update(pr); err != nil {
				return err
			}
		}
	}
	return committer.Commit()
}

// InsertReleases migrates release
func InsertReleases(rels ...*Release) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)

	for _, rel := range rels {
		if _, err := sess.NoAutoTime().Insert(rel); err != nil {
			return err
		}

		if len(rel.Attachments) > 0 {
			for i := range rel.Attachments {
				rel.Attachments[i].ReleaseID = rel.ID
			}

			if _, err := sess.NoAutoTime().Insert(rel.Attachments); err != nil {
				return err
			}
		}
	}

	return committer.Commit()
}

// UpdateMigrationsByType updates all migrated repositories' posterid from gitServiceType to replace originalAuthorID to posterID
func UpdateMigrationsByType(tp structs.GitServiceType, externalUserID string, userID int64) error {
	if err := issues_model.UpdateIssuesMigrationsByType(tp, externalUserID, userID); err != nil {
		return err
	}

	if err := issues_model.UpdateCommentsMigrationsByType(tp, externalUserID, userID); err != nil {
		return err
	}

	if err := UpdateReleasesMigrationsByType(tp, externalUserID, userID); err != nil {
		return err
	}

	if err := issues_model.UpdateReactionsMigrationsByType(tp, externalUserID, userID); err != nil {
		return err
	}
	return issues_model.UpdateReviewsMigrationsByType(tp, externalUserID, userID)
}
