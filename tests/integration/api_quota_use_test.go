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
	"strings"
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

func (e *quotaEnv) APIPathForRepo(uriFormat string, a ...any) string {
	path := fmt.Sprintf(uriFormat, a...)
	return fmt.Sprintf("/api/v1/repos/%s/%s%s", e.User.User.Name, e.Repo.Name, path)
}

func (e *quotaEnv) Cleanup() {
	for i := len(e.cleanups) - 1; i >= 0; i-- {
		e.cleanups[i]()
	}
}

func (e *quotaEnv) WithoutQuota(t *testing.T, task func()) {
	defer e.SetRuleLimit(t, "all", -1)()
	task()
}

func (e *quotaEnv) SetupWithSingleQuotaRule(t *testing.T) {
	t.Helper()

	cleaner := test.MockVariableValue(&setting.Quota.Enabled, true)
	e.cleanups = append(e.cleanups, cleaner)
	cleaner = test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())
	e.cleanups = append(e.cleanups, cleaner)

	// Create a default group
	cleaner = createQuotaGroup(t, "default")
	e.cleanups = append(e.cleanups, cleaner)

	// Create a single all-encompassing rule
	unlimited := int64(-1)
	ruleAll := api.CreateQuotaRuleOptions{
		Name:     "all",
		Limit:    &unlimited,
		Subjects: []string{"size:all"},
	}
	cleaner = createQuotaRule(t, ruleAll)
	e.cleanups = append(e.cleanups, cleaner)

	// Add these rules to the group
	cleaner = e.AddRuleToGroup(t, "default", "all")
	e.cleanups = append(e.cleanups, cleaner)

	// Add the user to the quota group
	cleaner = e.AddUserToGroup(t, "default", e.User.User.Name)
	e.cleanups = append(e.cleanups, cleaner)
}

func (e *quotaEnv) SetupWithMultipleQuotaRules(t *testing.T) {
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

func TestAPIQuotaEnforcement(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		testAPIQuotaEnforcement(t)
	})
}

func TestAPIQuotaCountsTowardsCorrectUser(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := prepareQuotaEnv(t, "quota-correct-user-test")
		defer env.Cleanup()
		env.SetupWithSingleQuotaRule(t)

		// Create a new group, with size:all set to 0
		defer createQuotaGroup(t, "limited")()
		zero := int64(0)
		defer createQuotaRule(t, api.CreateQuotaRuleOptions{
			Name:     "limited",
			Limit:    &zero,
			Subjects: []string{"size:all"},
		})()
		defer env.AddRuleToGroup(t, "limited", "limited")()

		// Add the admin user to it
		defer env.AddUserToGroup(t, "limited", env.Admin.User.Name)()

		// Add the admin user as collaborator to our repo
		perm := "admin"
		req := NewRequestWithJSON(t, "PUT",
			env.APIPathForRepo("/collaborators/%s", env.Admin.User.Name),
			api.AddCollaboratorOption{
				Permission: &perm,
			}).AddTokenAuth(env.User.Token)
		env.User.Session.MakeRequest(t, req, http.StatusNoContent)

		// Now, try to push something as admin!
		req = NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
			BranchName: "admin-branch",
		}).AddTokenAuth(env.Admin.Token)
		env.Admin.Session.MakeRequest(t, req, http.StatusCreated)
	})
}

func testAPIQuotaEnforcement(t *testing.T) {
	env := prepareQuotaEnv(t, "quota-enforcement")
	defer env.Cleanup()
	env.SetupWithSingleQuotaRule(t)

	t.Run("#/user/repos", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer env.SetRuleLimit(t, "all", 0)()

		t.Run("CREATE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithJSON(t, "POST", "/api/v1/user/repos", api.CreateRepoOption{
				Name:     "quota-exceeded",
				AutoInit: true,
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
		})

		t.Run("LIST", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/repos").AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusOK)
		})
	})

	// TODO
	t.Run("#/orgs/{org}/repos", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		t.Run("LIST", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
		})

		t.Run("CREATE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
		})
	})

	// TODO
	t.Run("#/repos/migrate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
	})

	// TODO
	t.Run("#/repos/{template_owner}/{template_repo}/generate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
	})

	t.Run("#/repos/{username}/{reponame}", func(t *testing.T) {
		// Lets create a new repo to play with.
		repo, _, repoCleanup := CreateDeclarativeRepoWithOptions(t, env.User.User, DeclarativeRepoOptions{})
		defer repoCleanup()

		// Drop the quota to 0
		defer env.SetRuleLimit(t, "all", 0)()

		t.Run("GET", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s", env.User.User.Name, repo.Name).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusOK)
		})
		t.Run("PATCH", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			desc := "Some description"
			req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s", env.User.User.Name, repo.Name), api.EditRepoOption{
				Description: &desc,
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusOK)
		})
		t.Run("DELETE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s", env.User.User.Name, repo.Name).
				AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusNoContent)
		})

		t.Run("branches", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Create a branch we can delete later
			env.WithoutQuota(t, func() {
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
					BranchName: "to-delete",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusCreated)
			})

			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", env.APIPathForRepo("/branches")).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/branches"), api.CreateBranchRepoOption{
					BranchName: "quota-exceeded",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("{branch}", func(t *testing.T) {
				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/branches/to-delete")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "DELETE", env.APIPathForRepo("/branches/to-delete")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusNoContent)
				})
			})
		})

		t.Run("contents", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			var fileSha string

			// Create a file to play with
			env.WithoutQuota(t, func() {
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/contents/plaything.txt"), api.CreateFileOptions{
					ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
				}).AddTokenAuth(env.User.Token)
				resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

				var r api.FileResponse
				DecodeJSON(t, resp, &r)

				fileSha = r.Content.SHA
			})

			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", env.APIPathForRepo("/contents")).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/contents"), api.ChangeFilesOptions{
					Files: []*api.ChangeFileOperation{
						{
							Operation: "create",
							Path:      "quota-exceeded.txt",
						},
					},
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("{filepath}", func(t *testing.T) {
				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/contents/plaything.txt")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("CREATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/contents/plaything.txt"), api.CreateFileOptions{
						ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
				t.Run("UPDATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "PUT", env.APIPathForRepo("/contents/plaything.txt"), api.UpdateFileOptions{
						ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
						DeleteFileOptions: api.DeleteFileOptions{
							SHA: fileSha,
						},
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					// Deleting a file fails, because it creates a new commit,
					// which would increase the quota use.
					req := NewRequestWithJSON(t, "DELETE", env.APIPathForRepo("/contents/plaything.txt"), api.DeleteFileOptions{
						SHA: fileSha,
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
			})
		})

		t.Run("diffpatch", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithJSON(t, "PUT", env.APIPathForRepo("/contents/README.md"), api.UpdateFileOptions{
				ContentBase64: base64.StdEncoding.EncodeToString([]byte("hello world")),
				DeleteFileOptions: api.DeleteFileOptions{
					SHA: "c0ffeebabe",
				},
			}).AddTokenAuth(env.User.Token)
			env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
		})

		// TODO
		t.Run("forks", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
			})
		})

		// TODO
		t.Run("mirror-sync", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
		})

		// TODO
		t.Run("issues", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			t.Run("comments/{id}/assets", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				t.Run("LIST", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
				})
				t.Run("CREATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
				})

				t.Run("{attachment_id}", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					t.Run("GET", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()
					})
					t.Run("DELETE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()
					})
					t.Run("UPDATE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()
					})
				})
			})

			t.Run("{index}/assets", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				t.Run("LIST", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
				})
				t.Run("CREATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
				})

				t.Run("{attachment_id}", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					t.Run("GET", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()
					})
					t.Run("DELETE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()
					})
					t.Run("UPDATE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()
					})
				})
			})
		})

		// TODO
		t.Run("pulls", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
			})

			t.Run("{index}", func(t *testing.T) {
				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
				})
				t.Run("UPDATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()
				})

				t.Run("merge", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					t.Run("GET", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()
					})
					t.Run("MERGE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()
					})
				})
			})
		})

		t.Run("releases", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			var releaseID int64

			// Create a release so that there's something to play with.
			env.WithoutQuota(t, func() {
				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/releases"), api.CreateReleaseOption{
					TagName: "play-release-tag",
					Title:   "play-release",
				}).AddTokenAuth(env.User.Token)
				resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

				var q api.Release
				DecodeJSON(t, resp, &q)

				releaseID = q.ID
			})

			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", env.APIPathForRepo("/releases")).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/releases"), api.CreateReleaseOption{
					TagName: "play-release-tag-two",
					Title:   "play-release-two",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("tags/{tag}", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Create a release for our subtests
				env.WithoutQuota(t, func() {
					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/releases"), api.CreateReleaseOption{
						TagName: "play-release-tag-subtest",
						Title:   "play-release-subtest",
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusCreated)
				})

				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/releases/tags/play-release-tag-subtest")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "DELETE", env.APIPathForRepo("/releases/tags/play-release-tag-subtest")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusNoContent)
				})
			})

			t.Run("{id}", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				var tmpReleaseID int64

				// Create a release so that there's something to play with.
				env.WithoutQuota(t, func() {
					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/releases"), api.CreateReleaseOption{
						TagName: "tmp-tag",
						Title:   "tmp-release",
					}).AddTokenAuth(env.User.Token)
					resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

					var q api.Release
					DecodeJSON(t, resp, &q)

					tmpReleaseID = q.ID
				})

				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/releases/%d", tmpReleaseID)).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("UPDATE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequestWithJSON(t, "PATCH", env.APIPathForRepo("/releases/%d", tmpReleaseID), api.EditReleaseOption{
						TagName: "tmp-tag-two",
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "DELETE", env.APIPathForRepo("/releases/%d", tmpReleaseID)).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusNoContent)
				})

				t.Run("assets", func(t *testing.T) {
					t.Run("LIST", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						req := NewRequest(t, "GET", env.APIPathForRepo("/releases/%d/assets", releaseID)).
							AddTokenAuth(env.User.Token)
						env.User.Session.MakeRequest(t, req, http.StatusOK)
					})
					t.Run("CREATE", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						body := strings.NewReader("hello world")
						req := NewRequestWithBody(t, "POST", env.APIPathForRepo("/releases/%d/assets?name=bar.txt", releaseID), body).
							AddTokenAuth(env.User.Token)
						req.Header.Add("Content-Type", "text/plain")
						env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
					})

					t.Run("{attachment_id}", func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						var attachmentID int64

						// Create an attachment to play with
						env.WithoutQuota(t, func() {
							body := strings.NewReader("hello world")
							req := NewRequestWithBody(t, "POST", env.APIPathForRepo("/releases/%d/assets?name=foo.txt", releaseID), body).
								AddTokenAuth(env.User.Token)
							req.Header.Add("Content-Type", "text/plain")
							resp := env.User.Session.MakeRequest(t, req, http.StatusCreated)

							var q api.Attachment
							DecodeJSON(t, resp, &q)

							attachmentID = q.ID
						})

						t.Run("GET", func(t *testing.T) {
							defer tests.PrintCurrentTest(t)()

							req := NewRequest(t, "GET", env.APIPathForRepo("/releases/%d/assets/%d", releaseID, attachmentID)).
								AddTokenAuth(env.User.Token)
							env.User.Session.MakeRequest(t, req, http.StatusOK)
						})
						t.Run("UPDATE", func(t *testing.T) {
							defer tests.PrintCurrentTest(t)()

							req := NewRequestWithJSON(t, "PATCH", env.APIPathForRepo("/releases/%d/assets/%d", releaseID, attachmentID), api.EditAttachmentOptions{
								Name: "new-name.txt",
							}).AddTokenAuth(env.User.Token)
							env.User.Session.MakeRequest(t, req, http.StatusCreated)
						})
						t.Run("DELETE", func(t *testing.T) {
							defer tests.PrintCurrentTest(t)()

							req := NewRequest(t, "DELETE", env.APIPathForRepo("/releases/%d/assets/%d", releaseID, attachmentID)).
								AddTokenAuth(env.User.Token)
							env.User.Session.MakeRequest(t, req, http.StatusNoContent)
						})
					})
				})
			})
		})

		t.Run("tags", func(t *testing.T) {
			t.Run("LIST", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", env.APIPathForRepo("/tags")).
					AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusOK)
			})
			t.Run("CREATE", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/tags"), api.CreateTagOption{
					TagName: "tag-quota-test",
				}).AddTokenAuth(env.User.Token)
				env.User.Session.MakeRequest(t, req, http.StatusRequestEntityTooLarge)
			})

			t.Run("{tag}", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				env.WithoutQuota(t, func() {
					req := NewRequestWithJSON(t, "POST", env.APIPathForRepo("/tags"), api.CreateTagOption{
						TagName: "tag-quota-test-2",
					}).AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusCreated)
				})

				t.Run("GET", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", env.APIPathForRepo("/tags/tag-quota-test-2")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusOK)
				})
				t.Run("DELETE", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "DELETE", env.APIPathForRepo("/tags/tag-quota-test-2")).
						AddTokenAuth(env.User.Token)
					env.User.Session.MakeRequest(t, req, http.StatusNoContent)
				})
			})
		})

		// TODO
		t.Run("transfer", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
		})
	})

	// TODO
	t.Run("#/packages/{owner}/{type}/{name}/{version}", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		t.Run("CREATE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
		})
		t.Run("GET", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
		})
		t.Run("DELETE", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
		})
	})
}

func TestAPIQuotaUserBasics(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := prepareQuotaEnv(t, "quota-enforcement")
		defer env.Cleanup()

		env.SetupWithMultipleQuotaRules(t)

		t.Run("quota usage change", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/user/quota").AddTokenAuth(env.User.Token)
			resp := env.User.Session.MakeRequest(t, req, http.StatusOK)

			var q api.QuotaInfo
			DecodeJSON(t, resp, &q)

			assert.Greater(t, q.Used.Size.Repos.Public, int64(0))

			t.Run("admin view", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequestf(t, "GET", "/api/v1/admin/users/%s/quota", env.User.User.Name).AddTokenAuth(env.Admin.Token)
				resp := env.Admin.Session.MakeRequest(t, req, http.StatusOK)

				var q api.QuotaInfoAdmin
				DecodeJSON(t, resp, &q)

				assert.Greater(t, q.Used.Size.Repos.Public, int64(0))
			})
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
