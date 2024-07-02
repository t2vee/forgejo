// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package integration

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	quota_model "code.gitea.io/gitea/models/quota"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

type quotaEnvUser struct {
	User    *user_model.User
	Session *TestSession
	Token   string
}

type quotaEnv struct {
	Admin quotaEnvUser
	User  quotaEnvUser
	Repo  *repo_model.Repository

	cleanups []func()
}

func (e *quotaEnv) APIPathForRepo(uri string) string {
	return fmt.Sprintf("/api/v1/repos/%s/%s/%s", e.User.User.Name, e.Repo.Name, uri)
}

func (e *quotaEnv) Cleanup() {
	for i := len(e.cleanups) - 1; i >= 0; i-- {
		e.cleanups[i]()
	}
}

func (e *quotaEnv) SetupQuotas(t *testing.T) {
	t.Helper()

	cleaner := test.MockVariableValue(&setting.Quota.Enabled, true)
	e.cleanups = append(e.cleanups, cleaner)
	cleaner = test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())
	e.cleanups = append(e.cleanups, cleaner)

	// Create a default group
	cleaner = createQuotaGroup(t, "default")
	e.cleanups = append(e.cleanups, cleaner)

	// Create three rules: all, repo-size, and asset-size
	zero := int64(0)
	ruleAll := api.CreateQuotaRuleOptions{
		Name:     "all",
		Limit:    &zero,
		Subjects: []string{"size:all"},
	}
	cleaner = createQuotaRule(t, ruleAll)
	e.cleanups = append(e.cleanups, cleaner)

	fifteenMb := int64(1024 * 1024 * 15)
	ruleRepoSize := api.CreateQuotaRuleOptions{
		Name:     "repo-size",
		Limit:    &fifteenMb,
		Subjects: []string{"size:repos:all"},
	}
	cleaner = createQuotaRule(t, ruleRepoSize)
	e.cleanups = append(e.cleanups, cleaner)

	ruleAssetSize := api.CreateQuotaRuleOptions{
		Name:     "asset-size",
		Limit:    &fifteenMb,
		Subjects: []string{"size:assets:all"},
	}
	cleaner = createQuotaRule(t, ruleAssetSize)
	e.cleanups = append(e.cleanups, cleaner)

	// Add these rules to the group
	cleaner = e.AddRuleToGroup(t, "default", "all")
	e.cleanups = append(e.cleanups, cleaner)
	cleaner = e.AddRuleToGroup(t, "default", "repo-size")
	e.cleanups = append(e.cleanups, cleaner)
	cleaner = e.AddRuleToGroup(t, "default", "asset-size")
	e.cleanups = append(e.cleanups, cleaner)

	// Add the user to the quota group
	cleaner = e.AddUserToGroup(t, "default", e.User.User.Name)
	e.cleanups = append(e.cleanups, cleaner)
}

func (e *quotaEnv) AddUserToGroup(t *testing.T, group, user string) func() {
	t.Helper()

	req := NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/%s/users/%s", group, user).AddTokenAuth(e.Admin.Token)
	e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)

	return func() {
		req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/%s/users/%s", group, user).AddTokenAuth(e.Admin.Token)
		e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)
	}
}

func (e *quotaEnv) SetRuleLimit(t *testing.T, rule string, limit int64) func() {
	t.Helper()

	originalRule, err := quota_model.GetRuleByName(db.DefaultContext, rule)
	assert.NoError(t, err)
	assert.NotNil(t, originalRule)

	req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/admin/quota/rules/%s", rule), api.EditQuotaRuleOptions{
		Limit: &limit,
	}).AddTokenAuth(e.Admin.Token)
	e.Admin.Session.MakeRequest(t, req, http.StatusOK)

	return func() {
		e.SetRuleLimit(t, rule, originalRule.Limit)
	}
}

func (e *quotaEnv) RemoveRuleFromGroup(t *testing.T, group, rule string) {
	t.Helper()

	req := NewRequestf(t, "DELETE", "/api/v1/admin/quota/groups/%s/rules/%s", group, rule).AddTokenAuth(e.Admin.Token)
	e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)
}

func (e *quotaEnv) AddRuleToGroup(t *testing.T, group, rule string) func() {
	t.Helper()

	req := NewRequestf(t, "PUT", "/api/v1/admin/quota/groups/%s/rules/%s", group, rule).AddTokenAuth(e.Admin.Token)
	e.Admin.Session.MakeRequest(t, req, http.StatusNoContent)

	return func() {
		e.RemoveRuleFromGroup(t, group, rule)
	}
}

func prepareQuotaEnv(t *testing.T, username string) *quotaEnv {
	t.Helper()

	env := quotaEnv{}

	// Set up the admin user
	env.Admin.User = unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
	env.Admin.Session = loginUser(t, env.Admin.User.Name)
	env.Admin.Token = getTokenForLoggedInUser(t, env.Admin.Session, auth_model.AccessTokenScopeAll)

	// Create a test user
	userCleanup := apiCreateUser(t, username)
	env.User.User = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: username})
	env.User.Session = loginUser(t, env.User.User.Name)
	env.User.Token = getTokenForLoggedInUser(t, env.User.Session, auth_model.AccessTokenScopeAll)
	env.cleanups = append(env.cleanups, userCleanup)

	// Create a repository
	repo, _, repoCleanup := CreateDeclarativeRepoWithOptions(t, env.User.User, DeclarativeRepoOptions{})
	env.Repo = repo
	env.cleanups = append(env.cleanups, repoCleanup)

	return &env
}

func TestAPIQuotaUserCleanSlate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer test.MockVariableValue(&setting.Quota.Enabled, true)()
		defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

		env := prepareQuotaEnv(t, "qt-clean-slate")
		defer env.Cleanup()

		t.Run("branch creation", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Create a branch
			req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
				BranchName: "branch-to-delete",
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusCreated)
		})
	})
}

func TestAPIQuotaUserExp(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := prepareQuotaEnv(t, "quota-enforcement")
		defer env.Cleanup()

		env.SetupQuotas(t)

		t.Run("quota usage change", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaInfo
			DecodeJSON(t, resp, &q)

			assert.Greater(t, q.Used.Size.Repos.Public, int64(0))
		})

		t.Run("quota check passing", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/check?subject=size:repos:all").AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusOK)

			var q bool
			DecodeJSON(t, resp, &q)

			assert.True(t, q)
		})

		t.Run("quota check failing after limit change", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer env.SetRuleLimit(t, "repo-size", 0)()

			req := NewRequest(t, "GET", "/api/v1/user/quota/check?subject=size:repos:all").AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusOK)

			var q bool
			DecodeJSON(t, resp, &q)

			assert.False(t, q)
		})

		t.Run("quota enforcement", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer env.SetRuleLimit(t, "repo-size", 0)()

			t.Run("repoCreateFile", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/contents/new-file.txt"), api.CreateFileOptions{
					ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("repoCreateBranch", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
					BranchName: "new-branch",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("repoDeleteBranch", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Temporarily disable quota checking
				defer env.SetRuleLimit(t, "repo-size", -1)()
				defer env.SetRuleLimit(t, "all", -1)()

				// Create a branch
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
					BranchName: "branch-to-delete",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusCreated)

				// Set the limit back. No need to defer, the first one will set it
				// back to the correct value.
				env.SetRuleLimit(t, "all", 0)
				env.SetRuleLimit(t, "repo-size", 0)

				// Deleting a branch does not incur quota enforcement
				req = NewRequest(t, "DELETE", env.APIPathForRepo("/branches/branch-to-delete")).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusNoContent)
			})
		})
	})
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
