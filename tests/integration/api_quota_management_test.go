// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package integration

import (
	"fmt"
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

func createQuotaGroup(t *testing.T, name string) func() {
	t.Helper()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups", api.CreateQuotaGroupOptions{
		Name: name,
	}).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusCreated)

	return func() {
		req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/%s", name).AddTokenAuth(adminToken)
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

	t.Run("adminCreateQuotaRule", func(t *testing.T) {
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

	t.Run("adminDeleteQuotaRule", func(t *testing.T) {
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

	t.Run("adminEditQuotaRule", func(t *testing.T) {
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

	t.Run("adminListQuotaRules", func(t *testing.T) {
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

func TestAPIQuotaAdminRoutesGroups(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	zero := int64(0)

	ruleDenyAll := api.CreateQuotaRuleOptions{
		Name:     "deny-all",
		Limit:    &zero,
		Subjects: []string{"size:all"},
	}

	username := "quota-test-user"
	defer apiCreateUser(t, username)()

	t.Run("adminCreateQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer createQuotaGroup(t, "default")()

		group, err := quota_model.GetGroupByName(db.DefaultContext, "default")
		assert.NoError(t, err)
		assert.Equal(t, "default", group.Name)
		assert.Len(t, group.Rules, 0)
	})

	t.Run("adminDeleteQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		createQuotaGroup(t, "default")

		req := NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		group, err := quota_model.GetGroupByName(db.DefaultContext, "default")
		assert.NoError(t, err)
		assert.Nil(t, group)
	})

	t.Run("adminAddRuleToQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaRule(t, ruleDenyAll)()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/default/rules", api.AddRuleToQuotaGroupOptions{
			Name: "deny-all",
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusCreated)

		group, err := quota_model.GetGroupByName(db.DefaultContext, "default")
		assert.NoError(t, err)
		assert.Len(t, group.Rules, 1)
		assert.Equal(t, "deny-all", group.Rules[0].Name)
	})

	t.Run("adminRemoveRuleFromQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaRule(t, ruleDenyAll)()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/default/rules", api.AddRuleToQuotaGroupOptions{
			Name: "deny-all",
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusCreated)

		req = NewRequest(t, "DELETE", "/api/v1/admin/quota/groups/default/rules/deny-all").AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		group, err := quota_model.GetGroupByName(db.DefaultContext, "default")
		assert.NoError(t, err)
		assert.Equal(t, "default", group.Name)
		assert.Empty(t, group.Rules)
	})

	t.Run("adminGetQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaRule(t, ruleDenyAll)()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/default/rules", api.AddRuleToQuotaGroupOptions{
			Name: "deny-all",
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusCreated)

		req = NewRequest(t, "GET", "/api/v1/admin/quota/groups/default").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaGroup
		DecodeJSON(t, resp, &q)

		assert.Equal(t, "default", q.Name)
		assert.Len(t, q.Rules, 1)
		assert.Equal(t, "deny-all", q.Rules[0].Name)
	})

	t.Run("adminListQuotaGroups", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaRule(t, ruleDenyAll)()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/default/rules", api.AddRuleToQuotaGroupOptions{
			Name: "deny-all",
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusCreated)

		req = NewRequest(t, "GET", "/api/v1/admin/quota/groups").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaGroupList
		DecodeJSON(t, resp, &q)

		assert.Len(t, q, 1)
		assert.Equal(t, "default", q[0].Name)
		assert.Len(t, q[0].Rules, 1)
		assert.Equal(t, "deny-all", q[0].Rules[0].Name)
	})

	t.Run("adminAddUserToQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/default/users", api.QuotaGroupAddOrRemoveUserOption{
			Username: username,
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusCreated)

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})

		groups, err := quota_model.GetGroupsForUser(db.DefaultContext, user.ID)
		assert.NoError(t, err)
		assert.Len(t, groups, 1)
		assert.Equal(t, "default", groups[0].Name)
	})

	t.Run("adminRemoveUserFromQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/default/users", api.QuotaGroupAddOrRemoveUserOption{
			Username: username,
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusCreated)

		req = NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/default/users/%s", username).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})
		groups, err := quota_model.GetGroupsForUser(db.DefaultContext, user.ID)
		assert.NoError(t, err)
		assert.Empty(t, groups)
	})

	t.Run("adminListUsersInQuotaGroup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()

		req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/default/users", api.QuotaGroupAddOrRemoveUserOption{
			Username: username,
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusCreated)

		req = NewRequest(t, "GET", "/api/v1/admin/quota/groups/default/users").AddTokenAuth(adminToken)
		resp := adminSession.MakeRequest(t, req, http.StatusOK)

		var q []api.User
		DecodeJSON(t, resp, &q)

		assert.Len(t, q, 1)
		assert.Equal(t, username, q[0].UserName)
	})

	t.Run("adminSetUserQuotaGroups", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer createQuotaGroup(t, "default")()
		defer createQuotaGroup(t, "test-1")()
		defer createQuotaGroup(t, "test-2")()

		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/admin/users/%s/quota/groups", username), api.SetUserQuotaGroupsOptions{
			Groups: &[]string{"default", "test-1", "test-2"},
		}).AddTokenAuth(adminToken)
		adminSession.MakeRequest(t, req, http.StatusNoContent)

		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})

		groups, err := quota_model.GetGroupsForUser(db.DefaultContext, user.ID)
		assert.NoError(t, err)
		assert.Len(t, groups, 3)
	})
}

func TestAPIQuotaUserRoutes(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Quota.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	adminSession := loginUser(t, admin.Name)
	adminToken := getTokenForLoggedInUser(t, adminSession, auth_model.AccessTokenScopeAll)

	// Create a test user
	username := "quota-test-user-routes"
	defer apiCreateUser(t, username)()
	session := loginUser(t, username)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeAll)

	// Set up rules & groups for the user
	defer createQuotaGroup(t, "user-routes-deny")()
	defer createQuotaGroup(t, "user-routes-1kb")()

	zero := int64(0)
	ruleDenyAll := api.CreateQuotaRuleOptions{
		Name:     "user-routes-deny-all",
		Limit:    &zero,
		Subjects: []string{"size:all"},
	}
	defer createQuotaRule(t, ruleDenyAll)()
	oneKb := int64(1024)
	rule1KbStuff := api.CreateQuotaRuleOptions{
		Name:     "user-routes-1kb",
		Limit:    &oneKb,
		Subjects: []string{"size:assets:attachments:releases", "size:assets:packages:all", "size:git:lfs"},
	}
	defer createQuotaRule(t, rule1KbStuff)()

	req := NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/user-routes-deny/rules", api.AddRuleToQuotaGroupOptions{
		Name: "user-routes-deny-all",
	}).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusCreated)
	req = NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/user-routes-1kb/rules", api.AddRuleToQuotaGroupOptions{
		Name: "user-routes-1kb",
	}).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusCreated)

	req = NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/user-routes-deny/users", api.QuotaGroupAddOrRemoveUserOption{
		Username: username,
	}).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusCreated)
	req = NewRequestWithJSON(t, "POST", "/api/v1/admin/quota/groups/user-routes-1kb/users", api.QuotaGroupAddOrRemoveUserOption{
		Username: username,
	}).AddTokenAuth(adminToken)
	adminSession.MakeRequest(t, req, http.StatusCreated)

	t.Run("userGetQuota", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(token)
		resp := session.MakeRequest(t, req, http.StatusOK)

		var q api.QuotaInfo
		DecodeJSON(t, resp, &q)

		assert.Len(t, q.Rules, 2)
	})
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
