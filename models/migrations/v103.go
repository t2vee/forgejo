// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package migrations

import (
	"xorm.io/xorm"
)

func addRequireSignedCommitsOnProtectedBranch(x *xorm.Engine) error {
	type ProtectedBranch struct {
		ID                        int64  `xorm:"pk autoincr"` 
		RequireSignedCommits      bool               `xorm:"NOT NULL DEFAULT false"`
	}

	return x.Sync2(new(ProtectedBranch))
}