// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"xorm.io/xorm"
)

func AddExternalURLColumnToAttachmentTable(x *xorm.Engine) error {
	type Attachment struct {
		ID          int64 `xorm:"pk autoincr"`
		ExternalURL string
	}
	return x.Sync(new(Attachment))
}
