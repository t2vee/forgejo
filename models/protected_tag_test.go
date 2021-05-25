// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsUserAllowed(t *testing.T) {
	assert.NoError(t, PrepareTestDatabase())

	pt := &ProtectedTag{}
	allowed, err := pt.IsUserAllowed(1)
	assert.NoError(t, err)
	assert.False(t, allowed)

	pt = &ProtectedTag{
		WhitelistUserIDs: []int64{1},
	}
	allowed, err = pt.IsUserAllowed(1)
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = pt.IsUserAllowed(2)
	assert.NoError(t, err)
	assert.False(t, allowed)

	pt = &ProtectedTag{
		WhitelistTeamIDs: []int64{1},
	}
	allowed, err = pt.IsUserAllowed(1)
	assert.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = pt.IsUserAllowed(2)
	assert.NoError(t, err)
	assert.True(t, allowed)

	pt = &ProtectedTag{
		WhitelistUserIDs: []int64{1},
		WhitelistTeamIDs: []int64{1},
	}
	allowed, err = pt.IsUserAllowed(1)
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = pt.IsUserAllowed(2)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestIsUserAllowedToControlTag(t *testing.T) {
	protectedTags := []*ProtectedTag{
		{
			NamePattern:      `gitea\z`,
			WhitelistUserIDs: []int64{1},
		},
		{
			NamePattern:      `\Av-`,
			WhitelistUserIDs: []int64{2},
		},
		{
			NamePattern: "release",
		},
	}

	cases := []struct {
		name    string
		userid  int64
		allowed bool
	}{
		{
			name:    "test",
			userid:  1,
			allowed: true,
		},
		{
			name:    "test",
			userid:  3,
			allowed: true,
		},
		{
			name:    "gitea",
			userid:  1,
			allowed: true,
		},
		{
			name:    "gitea",
			userid:  3,
			allowed: false,
		},
		{
			name:    "test-gitea",
			userid:  1,
			allowed: true,
		},
		{
			name:    "test-gitea",
			userid:  3,
			allowed: false,
		},
		{
			name:    "gitea-test",
			userid:  1,
			allowed: true,
		},
		{
			name:    "gitea-test",
			userid:  3,
			allowed: true,
		},
		{
			name:    "v-1",
			userid:  1,
			allowed: false,
		},
		{
			name:    "v-1",
			userid:  2,
			allowed: true,
		},
		{
			name:    "release",
			userid:  1,
			allowed: false,
		},
	}

	for n, c := range cases {
		isAllowed, err := IsUserAllowedToControlTag(protectedTags, c.name, c.userid)
		assert.NoError(t, err)
		assert.Equal(t, c.allowed, isAllowed, "case %d: error should match", n)
	}
}
