// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"code.gitea.io/gitea/models"

	"github.com/go-xorm/xorm"
)

func removeLingeringIndexStatus(x *xorm.Engine) error {

	var orphaned []*models.RepoIndexerStatus

	err := x.
		Join("LEFT OUTER", "`repository`", "`repository`.id = `repo_indexer_status`.repo_id").
		Where("`repository`.id is null").
		Find(&orphaned)
	if err != nil {
		return err
	}

	for _, o := range orphaned {
		if _, err = x.Delete(o); err != nil {
			return err
		}
	}

	return nil
}
