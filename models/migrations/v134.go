// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"xorm.io/xorm"
)

func refixMergeBase(x *xorm.Engine) error {
	type Repository struct {
		ID        int64 `xorm:"pk autoincr"`
		OwnerID   int64 `xorm:"UNIQUE(s) index"`
		OwnerName string
		LowerName string `xorm:"UNIQUE(s) INDEX NOT NULL"`
		Name      string `xorm:"INDEX NOT NULL"`
	}

	type PullRequest struct {
		ID         int64 `xorm:"pk autoincr"`
		Index      int64
		HeadRepoID int64 `xorm:"INDEX"`
		BaseRepoID int64 `xorm:"INDEX"`
		HeadBranch string
		BaseBranch string
		MergeBase  string `xorm:"VARCHAR(40)"`

		HasMerged      bool   `xorm:"INDEX"`
		MergedCommitID string `xorm:"VARCHAR(40)"`
	}

	limit := setting.Database.IterateBufferSize
	if limit <= 0 {
		limit = 50
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	count, err := x.Where("has_merged = ?", true).Count(new(PullRequest))
	if err != nil {
		return err
	}
	log.Info("%d Merged Pull Request(s) to migrate ...", count)

	i := 0
	start := 0
	for {
		prs := make([]PullRequest, 0, 50)
		if err := x.Limit(limit, start).Asc("id").Where("has_merged = ?", true).Find(&prs); err != nil {
			return fmt.Errorf("Find: %v", err)
		}
		if len(prs) == 0 {
			break
		}

		start += 50
		for _, pr := range prs {
			baseRepo := &Repository{ID: pr.BaseRepoID}
			has, err := x.Table("repository").Get(baseRepo)
			if err != nil {
				return fmt.Errorf("Unable to get base repo %d %v", pr.BaseRepoID, err)
			}
			if !has {
				log.Error("Missing base repo with id %d for PR ID %d", pr.BaseRepoID, pr.ID)
				continue
			}
			userPath := filepath.Join(setting.RepoRootPath, strings.ToLower(baseRepo.OwnerName))
			repoPath := filepath.Join(userPath, strings.ToLower(baseRepo.Name)+".git")

			gitRefName := fmt.Sprintf("refs/pull/%d/head", pr.Index)

			stdout := new(bytes.Buffer)
			err = git.NewCommand(git.DefaultContext, "rev-list", "--parents", "-n", "1", pr.MergedCommitID).RunWithContext(&git.RunContext{Dir: repoPath, Timeout: -1, Stdout: stdout})
			parentsString := stdout.String()
			if err != nil {
				log.Error("Unable to get parents for merged PR ID %d, Index %d in %s/%s. Error: %v", pr.ID, pr.Index, baseRepo.OwnerName, baseRepo.Name, err)
				continue
			}
			parents := strings.Split(strings.TrimSpace(parentsString), " ")
			if len(parents) < 3 {
				continue
			}

			// we should recalculate
			args := append([]string{"merge-base", "--"}, parents[1:]...)
			args = append(args, gitRefName)

			stdout.Reset()
			err = git.NewCommand(git.DefaultContext, args...).RunWithContext(&git.RunContext{Dir: repoPath, Timeout: -1, Stdout: stdout})
			pr.MergeBase = stdout.String()
			if err != nil {
				log.Error("Unable to get merge base for merged PR ID %d, Index %d in %s/%s. Error: %v", pr.ID, pr.Index, baseRepo.OwnerName, baseRepo.Name, err)
				continue
			}
			pr.MergeBase = strings.TrimSpace(pr.MergeBase)
			x.ID(pr.ID).Cols("merge_base").Update(pr)
			i++
			select {
			case <-ticker.C:
				log.Info("%d/%d (%2.0f%%) Pull Request(s) migrated in %d batches. %d PRs Remaining ...", i, count, float64(i)/float64(count)*100, int(math.Ceil(float64(i)/float64(limit))), count-int64(i))
			default:
			}
		}
	}

	log.Info("Completed migrating %d Pull Request(s) in: %d batches", count, int(math.Ceil(float64(i)/float64(limit))))
	return nil
}
