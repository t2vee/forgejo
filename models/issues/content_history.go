// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/avatars"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"

	"xorm.io/builder"
)

// ContentHistory save issue/comment content history revisions.
type ContentHistory struct {
	ID             int64 `xorm:"pk autoincr"`
	PosterID       int64
	IssueID        int64              `xorm:"INDEX"`
	CommentID      int64              `xorm:"INDEX"`
	EditedUnix     timeutil.TimeStamp `xorm:"INDEX"`
	ContentText    string             `xorm:"LONGTEXT"`
	IsFirstCreated bool
	IsDeleted      bool
}

// TableName provides the real table name
func (m *ContentHistory) TableName() string {
	return "issue_content_history"
}

func init() {
	db.RegisterModel(new(ContentHistory))
}

// SaveIssueContentHistory save history
func SaveIssueContentHistory(e db.Engine, posterID, issueID, commentID int64, editTime timeutil.TimeStamp, contentText string, isFirstCreated bool) error {
	ch := &ContentHistory{
		PosterID:       posterID,
		IssueID:        issueID,
		CommentID:      commentID,
		ContentText:    contentText,
		EditedUnix:     editTime,
		IsFirstCreated: isFirstCreated,
	}
	_, err := e.Insert(ch)
	if err != nil {
		log.Error("can not save issue content history. err=%v", err)
		return err
	}
	return nil
}

// QueryIssueContentHistoryEditedCountMap query related history count of each comment (comment_id = 0 means the main issue)
// only return the count map for "edited" (history revision count > 1) issues or comments.
func QueryIssueContentHistoryEditedCountMap(dbCtx context.Context, issueID int64) (map[int64]int, error) {
	type HistoryCountRecord struct {
		CommentID    int64
		HistoryCount int
	}
	records := make([]*HistoryCountRecord, 0)

	err := db.GetEngine(dbCtx).Select("comment_id, COUNT(1) as history_count").
		Table("issue_content_history").
		Where(builder.Eq{"issue_id": issueID}).
		GroupBy("comment_id").
		Having("history_count > 1").
		Find(&records)
	if err != nil {
		log.Error("can not query issue content history count map. err=%v", err)
		return nil, err
	}

	res := map[int64]int{}
	for _, r := range records {
		res[r.CommentID] = r.HistoryCount
	}
	return res, nil
}

// IssueContentListItem the list for web ui
type IssueContentListItem struct {
	UserID          int64
	UserName        string
	UserAvatar      string
	UserAvatarEmail string
	UseCustomAvatar bool
	UserAvatarLink  string

	HistoryID      int64
	EditedUnix     timeutil.TimeStamp
	IsFirstCreated bool
	IsDeleted      bool
}

// FetchIssueContentHistoryList fetch list
func FetchIssueContentHistoryList(dbCtx context.Context, issueID int64, commentID int64) ([]*IssueContentListItem, error) {
	res := make([]*IssueContentListItem, 0)
	err := db.GetEngine(dbCtx).Select("u.id as user_id, u.name as user_name,"+
		"u.avatar as user_avatar, u.avatar_email as user_avatar_email, u.use_custom_avatar,"+
		"h.id as history_id, h.edited_unix, h.is_first_created, h.is_deleted").
		Table([]string{"issue_content_history", "h"}).
		Join("INNER", []string{"user", "u"}, "h.poster_id = u.id").
		Where(builder.Eq{"issue_id": issueID, "comment_id": commentID}).
		OrderBy("edited_unix DESC").
		Find(&res)

	if err != nil {
		log.Error("can not fetch issue content history list. err=%v", err)
		return nil, err
	}

	for _, item := range res {
		item.UserAvatarLink = avatars.GenerateUserAvatarFastLink(item.UserName, 0)
	}
	return res, nil
}

//SoftDeleteIssueContentHistory soft delete
func SoftDeleteIssueContentHistory(dbCtx context.Context, historyID int64) error {
	if _, err := db.GetEngine(dbCtx).ID(historyID).Cols("is_deleted", "content_text").Update(&ContentHistory{
		IsDeleted:   true,
		ContentText: "",
	}); err != nil {
		log.Error("failed to soft delete issue content history. err=%v", err)
		return err
	}
	return nil
}

// ErrIssueContentHistoryNotExist not exist error
type ErrIssueContentHistoryNotExist struct {
	ID int64
}

// Error error string
func (err ErrIssueContentHistoryNotExist) Error() string {
	return fmt.Sprintf("issue content history does not exist [id: %d]", err.ID)
}

// GetIssueContentHistoryByID get issue content history
func GetIssueContentHistoryByID(dbCtx context.Context, id int64) (*ContentHistory, error) {
	h := &ContentHistory{}
	has, err := db.GetEngine(dbCtx).ID(id).Get(h)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrIssueContentHistoryNotExist{id}
	}
	return h, nil
}

// GetIssueContentHistoryAndPrev get a history and the previous non-deleted history (to compare)
func GetIssueContentHistoryAndPrev(dbCtx context.Context, id int64) (history, prevHistory *ContentHistory, err error) {
	history = &ContentHistory{}
	has, err := db.GetEngine(dbCtx).ID(id).Get(history)
	if err != nil {
		log.Error("failed to get issue content history %v. err=%v", id, err)
		return nil, nil, err
	} else if !has {
		log.Error("issue content history does not exist. id=%v. err=%v", id, err)
		return nil, nil, &ErrIssueContentHistoryNotExist{id}
	}

	prevHistory = &ContentHistory{}
	has, err = db.GetEngine(dbCtx).Where(builder.Eq{"issue_id": history.IssueID, "comment_id": history.CommentID, "is_deleted": false}).
		And(builder.Lt{"edited_unix": history.EditedUnix}).
		OrderBy("edited_unix DESC").Limit(1).
		Get(prevHistory)

	if err != nil {
		log.Error("failed to get issue content history %v. err=%v", id, err)
		return nil, nil, err
	} else if !has {
		return history, nil, nil
	}

	return history, prevHistory, nil
}
