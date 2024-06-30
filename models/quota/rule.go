// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package quota

import (
	"context"
	"slices"

	"code.gitea.io/gitea/models/db"
)

type Rule struct {
	Name     string         `xorm:"pk not null" json:"name"`
	Limit    int64          `xorm:"NOT NULL" binding:"Required" json:"limit"`
	Subjects []LimitSubject `json:"subjects,omitempty"`
}

func (r *Rule) TableName() string {
	return "quota_rule"
}

var indirectMap = map[LimitSubject]LimitSubjects{
	LimitSubjectSizeAll: {
		LimitSubjectSizeReposAll,
		LimitSubjectSizeReposPublic,
		LimitSubjectSizeReposPrivate,
		LimitSubjectSizeGitAll,
		LimitSubjectSizeGitLFS,
		LimitSubjectSizeAssetsAll,
		LimitSubjectSizeAssetsAttachmentsAll,
		LimitSubjectSizeAssetsAttachmentsIssues,
		LimitSubjectSizeAssetsArtifacts,
		LimitSubjectSizeAssetsPackagesAll,
		LimitSubjectSizeWiki,
	},
	LimitSubjectSizeReposAll: {
		LimitSubjectSizeAll,
		LimitSubjectSizeReposPublic,
		LimitSubjectSizeReposPrivate,
	},
	LimitSubjectSizeReposPublic: {
		LimitSubjectSizeAll,
		LimitSubjectSizeReposAll,
	},
	LimitSubjectSizeReposPrivate: {
		LimitSubjectSizeAll,
		LimitSubjectSizeReposAll,
	},
	LimitSubjectSizeGitAll: {
		LimitSubjectSizeAll,
		LimitSubjectSizeReposAll,
		LimitSubjectSizeReposPublic,
		LimitSubjectSizeReposPrivate,
		LimitSubjectSizeGitLFS,
	},
	LimitSubjectSizeGitLFS: {
		LimitSubjectSizeAll,
		LimitSubjectSizeGitAll,
	},
	LimitSubjectSizeAssetsAll: {
		LimitSubjectSizeAll,
		LimitSubjectSizeAssetsAttachmentsAll,
		LimitSubjectSizeAssetsAttachmentsIssues,
		LimitSubjectSizeAssetsAttachmentsReleases,
		LimitSubjectSizeAssetsArtifacts,
		LimitSubjectSizeAssetsPackagesAll,
	},
	LimitSubjectSizeAssetsAttachmentsAll: {
		LimitSubjectSizeAll,
		LimitSubjectSizeAssetsAll,
		LimitSubjectSizeAssetsAttachmentsIssues,
		LimitSubjectSizeAssetsAttachmentsReleases,
	},
	LimitSubjectSizeAssetsAttachmentsIssues: {
		LimitSubjectSizeAll,
		LimitSubjectSizeAssetsAll,
		LimitSubjectSizeAssetsAttachmentsAll,
	},
	LimitSubjectSizeAssetsAttachmentsReleases: {
		LimitSubjectSizeAll,
		LimitSubjectSizeAssetsAll,
		LimitSubjectSizeAssetsAttachmentsAll,
	},
	LimitSubjectSizeAssetsArtifacts: {
		LimitSubjectSizeAll,
		LimitSubjectSizeAssetsAll,
	},
	LimitSubjectSizeAssetsPackagesAll: {
		LimitSubjectSizeAll,
		LimitSubjectSizeAssetsAll,
	},
	LimitSubjectSizeWiki: {
		LimitSubjectSizeAll,
	},
}

func (r Rule) Evaluate(used Used, forSubject LimitSubject) (bool, bool) {
	// If there's no limit, short circuit out
	if r.Limit == -1 {
		return true, true
	}

	// If evaluating against a subject the rule directly covers, return that
	if slices.Contains(r.Subjects, forSubject) {
		return used.CalculateFor(forSubject) <= r.Limit, true
	}

	// If evaluating against a subject the rule does not directly cover, check
	// if we have any rule that covers it indirectly.
	result := true
	var found bool
	for _, subject := range indirectMap[forSubject] {
		if !slices.Contains(r.Subjects, subject) {
			continue
		}

		ok, has := r.Evaluate(used, subject)
		if !has {
			continue
		}
		found = true
		result = result && ok
	}
	return result, found
}

func (r *Rule) Edit(ctx context.Context, limit *int64, subjects *LimitSubjects) error {
	cols := []string{}

	if limit != nil {
		r.Limit = *limit
		cols = append(cols, "limit")
	}
	if subjects != nil {
		r.Subjects = *subjects
		cols = append(cols, "subjects")
	}

	_, err := db.GetEngine(ctx).Where("name = ?", r.Name).Cols(cols...).Update(r)
	return err
}

func GetRuleByName(ctx context.Context, name string) (*Rule, error) {
	var rule Rule
	has, err := db.GetEngine(ctx).Where("name = ?", name).Get(&rule)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return &rule, err
}

func ListRules(ctx context.Context) ([]Rule, error) {
	var rules []Rule
	err := db.GetEngine(ctx).Find(&rules)
	return rules, err
}

func CreateRule(ctx context.Context, name string, limit int64, subjects LimitSubjects) error {
	_, err := db.GetEngine(ctx).Insert(Rule{
		Name:     name,
		Limit:    limit,
		Subjects: subjects,
	})
	return err
}

func DeleteRuleByName(ctx context.Context, name string) error {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	_, err = db.GetEngine(ctx).Delete(GroupRuleMapping{
		RuleName: name,
	})
	if err != nil {
		return err
	}

	_, err = db.GetEngine(ctx).Delete(Group{Name: name})
	return err
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
