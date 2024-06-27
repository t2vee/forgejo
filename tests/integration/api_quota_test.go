// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package integration

import (
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	quota_service "code.gitea.io/gitea/services/quota"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIQuotaDisabled(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, false)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	session := loginUser(t, user.Name)

	req := NewRequest(t, "GET", "/api/v1/user/quota")
	session.MakeRequest(t, req, http.StatusNotFound)
}

func apiCreateUser(t *testing.T, username string) func() {
	t.Helper()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	session := loginUser(t, admin.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	mustChangePassword := false
	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/users", api.CreateUserOption{
		Email:              "api+" + username + "@example.com",
		Username:           username,
		Password:           "password",
		MustChangePassword: &mustChangePassword,
	}).AddTokenAuth(token)
	session.MakeRequest(t, req, http.StatusCreated)

	return func() {
		req := NewRequest(t, "DELETE", "/api/v1/admin/users/"+username).AddTokenAuth(token)
		session.MakeRequest(t, req, http.StatusNoContent)
	}
}

func TestAPIQuotaEmptyUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	username := "quota-empty-user"
	defer apiCreateUser(t, username)()
	session := loginUser(t, username)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	t.Run("/user/quota", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(token)
		resp := session.MakeRequest(t, req, http.StatusOK)

		var q quota_service.UserQuota
		DecodeJSON(t, resp, &q)

		assert.EqualValues(t, quota_service.QuotaLimits{}, q.Limits)
		assert.EqualValues(t, 0, q.Used.Git.Code)
		assert.EqualValues(t, 0, q.Used.Git.LFS)
		assert.EqualValues(t, 0, q.Used.Assets.Attachments.Issues)
		assert.EqualValues(t, 0, q.Used.Assets.Attachments.Releases)
		assert.EqualValues(t, 0, q.Used.Assets.Artifacts)
		assert.EqualValues(t, 0, q.Used.Assets.Packages)

		t.Run("/user/quota/artifacts", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/artifacts").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedArtifactList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})

		t.Run("/user/quota/attachments", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/attachments").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedAttachmentList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})

		t.Run("/user/quota/packages", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/packages").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedPackageList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})
	})
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
