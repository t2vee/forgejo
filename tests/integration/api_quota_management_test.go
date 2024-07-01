// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package integration

import (
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	quota_model "code.gitea.io/gitea/models/quota"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
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

func TestAPIQuotaEmptyState(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	username := "quota-empty-user"
	defer apiCreateUser(t, username)()
	session := loginUser(t, username)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	t.Run("#/admin/users/quota-empty-user/quota", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
		adminSession := loginUser(t, admin.Name)
		adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

		req := NewRequest(t, "GET", "/api/v1/admin/users/quota-empty-user/quota").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaInfoAdmin
		DecodeJSON(t, resp, &q)

		assert.EqualValues(t, q.Used, api.QuotaUsed{})
		assert.Empty(t, q.Groups)
	})

	t.Run("#/user/quota", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(token)
		resp := session.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaInfo
		DecodeJSON(t, resp, &q)

		assert.EqualValues(t, q.Used, api.QuotaUsed{})
		assert.Empty(t, q.Rules)

		t.Run("#/user/quota/artifacts", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/artifacts").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedArtifactList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})

		t.Run("#/user/quota/attachments", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/attachments").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedAttachmentList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})

		t.Run("#/user/quota/packages", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/packages").AddTokenAuth(token)
			resp := session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaUsedPackageList
			DecodeJSON(t, resp, &q)

			assert.Empty(t, q)
		})
	})
}

func createQuotaRule(t *testing.T, opts api.CreateQuotaRuleOptions) func() {
	t.Helper()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/rules", opts).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusCreated)

	return func() {
		req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/rules/%s", opts.Name).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)
	}
}

func TestAPIQuotaAdminRoutesRules(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	zero := int64(0)
	oneKb := int64(1024)

	t.Run("adminCreateRule", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "deny-all",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})()

		rule, err := quota_model.GetRuleByName(db.DefaultContext, "deny-all")
		assert.NoError(t, err)
		assert.EqualValues(t, 0, rule.Limit)
	})

	t.Run("adminDeleteRule", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "deny-all",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})

		req := NewRequest(t, "DELETE", "/api/v1/admin/quota/rules/deny-all").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		rule, err := quota_model.GetRuleByName(db.DefaultContext, "deny-all")
		assert.NoError(t, err)
		assert.Nil(t, rule)
	})

	t.Run("adminEditRule", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "deny-all",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})()

		req := NewRequestWithJSON(t, "PATCH", "/api/v1/admin/quota/rules/deny-all", api.EditQuotaRuleOptions{
			Limit: &oneKb,
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		rule, err := quota_model.GetRuleByName(db.DefaultContext, "deny-all")
		assert.NoError(t, err)
		assert.EqualValues(t, 1024, rule.Limit)
	})

	t.Run("adminListRules", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "deny-all",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})()

		req := NewRequest(t, "GET", "/api/v1/admin/quota/rules").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var rules []api.QuotaRuleInfo
		DecodeJSON(t, resp, &rules)

		assert.Len(t, rules, 1)
		assert.Equal(t, "deny-all", rules[0].Name)
		assert.EqualValues(t, 0, rules[0].Limit)
	})
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
