// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"testing"

	"code.gitea.io/gitea/models/migrations/base"
)

func TestMain(m *testing.M) {
	base.MainTest(m)
}
