// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package integration

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	org_model "code.gitea.io/gitea/models/organization"
	quota_model "code.gitea.io/gitea/models/quota"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

	gouuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWebQuotaEnforcementRepoMigrate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		env.RunVisitAndPostToPageTests(t, "/repo/migrate", &Payload{
			"repo_name":  "migration-test",
			"clone_addr": env.Users.Limited.Repo.Link() + ".git",
		}, http.StatusOK)
	})
}

func TestWebQuotaEnforcementRepoCreate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		env.RunVisitAndPostToPageTests(t, "/repo/create", nil, http.StatusOK)
	})
}

func TestWebQuotaEnforcementRepoFork(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		page := fmt.Sprintf("%s/fork", env.Users.Limited.Repo.Link())
		env.RunVisitAndPostToPageTests(t, page, &Payload{
			"repo_name": "fork-test",
		}, http.StatusSeeOther)
	})
}

func TestWebQuotaEnforcementIssueAttachment(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		// Uploading to our repo => 413
		env.As(t, env.Users.Limited).
			With(Context{Repo: env.Users.Limited.Repo}).
			CreateAttachment("test.txt").
			ExpectStatus(http.StatusRequestEntityTooLarge)

		// Uploading to the limited org repo => 413
		env.As(t, env.Users.Limited).
			With(Context{Repo: env.Orgs.Limited.Repo}).
			CreateAttachment("test.txt").
			ExpectStatus(http.StatusRequestEntityTooLarge)

		// Uploading to the unlimited org repo => 200
		env.As(t, env.Users.Limited).
			With(Context{Repo: env.Orgs.Unlimited.Repo}).
			CreateAttachment("test.txt").
			ExpectStatus(http.StatusOK)
	})
}

func TestWebQuotaEnforcementMirrorSync(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		var mirrorRepo *repo_model.Repository

		env.As(t, env.Users.Limited).
			WithoutQuota(func(ctx *quotaWebEnvAsContext) {
				mirrorRepo = ctx.CreateMirror()
			}).
			With(Context{
				Repo:    mirrorRepo,
				Payload: &Payload{"action": "mirror-sync"},
			}).
			PostToPage(mirrorRepo.Link() + "/settings").
			ExpectStatus(http.StatusOK).
			ExpectFlashMessage("Quota exceeded, not pulling changes.")
	})
}

func TestWebQuotaEnforcementRepoContentEditing(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		// We're only going to test the GET requests here, because the entire combo
		// is covered by a route check.

		// Lets create a helper!
		runCheck := func(t *testing.T, path string, successStatus int) {
			t.Run("#"+path, func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Uploading to a limited user's repo => 413
				env.As(t, env.Users.Limited).
					VisitPage(env.Users.Limited.Repo.Link() + path).
					ExpectStatus(http.StatusRequestEntityTooLarge)

				// Limited org => 413
				env.As(t, env.Users.Limited).
					VisitPage(env.Orgs.Limited.Repo.Link() + path).
					ExpectStatus(http.StatusRequestEntityTooLarge)

				// Unlimited org => 200
				env.As(t, env.Users.Limited).
					VisitPage(env.Orgs.Unlimited.Repo.Link() + path).
					ExpectStatus(successStatus)
			})
		}

		paths := []string{
			"/_new/main",
			"/_edit/main/README.md",
			"/_delete/main",
			"/_upload/main",
			"/_diffpatch/main",
		}

		for _, path := range paths {
			runCheck(t, path, http.StatusOK)
		}

		// Run another check for `_cherrypick`. It's cumbersome to dig out a valid
		// commit id, so we'll use a fake, and treat 404 as a success: it's not 413,
		// and that's all we care about for this test.
		runCheck(t, "/_cherrypick/92cfceb39d57d914ed8b14d0e37643de0797ae56/main", http.StatusNotFound)
	})
}

func TestWebQuotaEnforcementRepoBranches(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		t.Run("create", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			csrfPath := env.Users.Limited.Repo.Link()

			runTest := func(t *testing.T, path string) {
				t.Run("#"+path, func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					env.As(t, env.Users.Limited).
						With(Context{
							Payload:  &Payload{"new_branch_name": "quota"},
							CSRFPath: &csrfPath,
						}).
						PostToRepoPage("/branches/_new" + path).
						ExpectStatus(http.StatusRequestEntityTooLarge)

					env.As(t, env.Users.Limited).
						With(Context{
							Payload:  &Payload{"new_branch_name": "quota"},
							CSRFPath: &csrfPath,
							Repo:     env.Orgs.Limited.Repo,
						}).
						PostToRepoPage("/branches/_new" + path).
						ExpectStatus(http.StatusRequestEntityTooLarge)

					env.As(t, env.Users.Limited).
						With(Context{
							Payload:  &Payload{"new_branch_name": "quota"},
							CSRFPath: &csrfPath,
							Repo:     env.Orgs.Unlimited.Repo,
						}).
						PostToRepoPage("/branches/_new" + path).
						ExpectStatus(http.StatusNotFound)
				})
			}

			// We're testing the first two against things that don't exist, so that
			// all three consistently return 404 if no quota enforcement happens.
			runTest(t, "/branch/no-such-branch")
			runTest(t, "/tag/no-such-tag")
			runTest(t, "/commit/92cfceb39d57d914ed8b14d0e37643de0797ae56")
		})

		t.Run("delete & restore", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			csrfPath := env.Users.Limited.Repo.Link()

			env.As(t, env.Users.Limited).
				With(Context{CSRFPath: &csrfPath}).
				WithoutQuota(func(ctx *quotaWebEnvAsContext) {
					ctx.With(Context{Payload: &Payload{"new_branch_name": "to-delete"}}).
						PostToRepoPage("/branches/_new/branch/main").
						ExpectStatus(http.StatusSeeOther)
				})

			env.As(t, env.Users.Limited).
				With(Context{CSRFPath: &csrfPath}).
				PostToRepoPage("/branches/delete?name=to-delete").
				ExpectStatus(http.StatusOK)

			env.As(t, env.Users.Limited).
				With(Context{CSRFPath: &csrfPath}).
				PostToRepoPage("/branches/restore?name=to-delete").
				ExpectStatus(http.StatusOK)
		})
	})
}

/*
Done:

  PR repo.Create => should filter out invalid targets
  DONE repo.CreatePost

  PR repo.Migrate => should filter out invalid targets
  DONE repo.MigratePost

  PR & DONE repo.Fork{,Post} => filter & target check

  DONE repo.UploadIssueAttachment => route check

  DONE mirror-sync

  DONE repo.DiffPreviewPost => applies a patch, route check
  DONE repo.EditFile, repo.NewFile, repo.NewFilePost, repo.UploadFilePost => route check
  DONE repo.NewDiffPatch, repo.CherryPick => route check

  DONE repo.MigrateRetryPost -> needs quota (done, need test)
  DONE repo.MigrateCancelPost -> doesn't need quota

  DONE /branches/_new => always quota check
       incl: repo.CreateBranch
  DONE repo.RestoreBranch{,Post} => no quota check needed. this does not change the storage size: it just restores what's already there.

  LATER repo.UploadFileToServer => where does this upload it to? quota check needed, but what subject?

  DONE user.PackageSettingsPost => no verification needed, packages can only be assigned to repos that have the same owner as the package, so no quota usage transfer is involved.

TODO:

  protected tags? do they ++ git size, or are they db only like protected branch settings?

  repo_lfs.LFSAutoAssociate? do we care?

  repo.ActionTransfer() => quota!

  repo.NewRelease{,Post} => route check? since it can create tags.
  repo.EditRelease{,Post} => same

  repo.UploadReleaseAttachment => route check

  repo.WikiPost => route check

  repo.MergePullRequest => route check

  repo.LastCommit POST => ???

  lfs.BatchHandler => special quota check
  lfs.UploadHandler => route or special?
*/

/**********************
 * Here be dragons!   *
 *                    *
 *      .             *
 *  .>   )\;`a__      *
 * (  _ _)/ /-." ~~   *
 *  `( )_ )/          *
 *  <_  <_ sb/dwb     *
 **********************/

type quotaWebEnv struct {
	Users quotaWebEnvUsers
	Orgs  quotaWebEnvOrgs

	cleaners []func()
}

type quotaWebEnvUsers struct {
	Limited quotaWebEnvUser
}

type quotaWebEnvOrgs struct {
	Limited   quotaWebEnvOrg
	Unlimited quotaWebEnvOrg
}

type quotaWebEnvOrg struct {
	Org *org_model.Organization

	Repo *repo_model.Repository

	QuotaGroup *quota_model.Group
	QuotaRule  *quota_model.Rule
}

type quotaWebEnvUser struct {
	User    *user_model.User
	Session *TestSession
	Repo    *repo_model.Repository

	QuotaGroup *quota_model.Group
	QuotaRule  *quota_model.Rule
}

type Payload map[string]string

type quotaWebEnvAsContext struct {
	t *testing.T

	Doer *quotaWebEnvUser
	Repo *repo_model.Repository

	Payload Payload

	CSRFPath *string

	request  *RequestWrapper
	response *httptest.ResponseRecorder
}

type Context struct {
	Repo     *repo_model.Repository
	Payload  *Payload
	CSRFPath *string
}

func (ctx *quotaWebEnvAsContext) With(opts Context) *quotaWebEnvAsContext {
	if opts.Repo != nil {
		ctx.Repo = opts.Repo
	}
	if opts.Payload != nil {
		for key, value := range *opts.Payload {
			ctx.Payload[key] = value
		}
	}
	if opts.CSRFPath != nil {
		ctx.CSRFPath = opts.CSRFPath
	}
	return ctx
}

func (ctx *quotaWebEnvAsContext) VisitPage(page string) *quotaWebEnvAsContext {
	ctx.t.Helper()

	ctx.request = NewRequest(ctx.t, "GET", page)

	return ctx
}

func (ctx *quotaWebEnvAsContext) ExpectStatus(status int) *quotaWebEnvAsContext {
	ctx.t.Helper()

	ctx.response = ctx.Doer.Session.MakeRequest(ctx.t, ctx.request, status)

	return ctx
}

func (ctx *quotaWebEnvAsContext) ExpectFlashMessage(value string) {
	ctx.t.Helper()

	htmlDoc := NewHTMLParser(ctx.t, ctx.response.Body)
	flashMessage := strings.TrimSpace(htmlDoc.Find(`.flash-message`).Text())

	assert.EqualValues(ctx.t, value, flashMessage)
}

func (ctx *quotaWebEnvAsContext) PostToPage(page string) *quotaWebEnvAsContext {
	ctx.t.Helper()

	payload := ctx.Payload
	csrfPath := page
	if ctx.CSRFPath != nil {
		csrfPath = *ctx.CSRFPath
	}

	payload["_csrf"] = GetCSRF(ctx.t, ctx.Doer.Session, csrfPath)

	ctx.request = NewRequestWithValues(ctx.t, "POST", page, payload)

	return ctx
}

func (ctx *quotaWebEnvAsContext) PostToRepoPage(page string) *quotaWebEnvAsContext {
	ctx.t.Helper()

	return ctx.PostToPage(ctx.Repo.Link() + page)
}

func (ctx *quotaWebEnvAsContext) CreateAttachment(filename string) *quotaWebEnvAsContext {
	ctx.t.Helper()

	body := &bytes.Buffer{}
	image := generateImg()

	// Setup multi-part
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	assert.NoError(ctx.t, err)
	_, err = io.Copy(part, &image)
	assert.NoError(ctx.t, err)
	err = writer.Close()
	assert.NoError(ctx.t, err)

	csrf := GetCSRF(ctx.t, ctx.Doer.Session, ctx.Repo.Link())

	ctx.request = NewRequestWithBody(ctx.t, "POST", ctx.Repo.Link()+"/issues/attachments", body)
	ctx.request.Header.Add("X-Csrf-Token", csrf)
	ctx.request.Header.Add("Content-Type", writer.FormDataContentType())

	return ctx
}

func (ctx *quotaWebEnvAsContext) WithoutQuota(task func(ctx *quotaWebEnvAsContext)) *quotaWebEnvAsContext {
	ctx.t.Helper()

	defer ctx.Doer.SetQuota(-1)()
	task(ctx)

	return ctx
}

func (ctx *quotaWebEnvAsContext) CreateMirror() *repo_model.Repository {
	ctx.t.Helper()

	doer := ctx.Doer.User

	repo, err := repo_service.CreateRepositoryDirectly(db.DefaultContext, doer, doer, repo_service.CreateRepoOptions{
		Name:     "test-mirror",
		IsMirror: true,
		Status:   repo_model.RepositoryBeingMigrated,
	})
	assert.NoError(ctx.t, err)

	return repo
}

func (user *quotaWebEnvUser) SetQuota(limit int64) func() {
	previousLimit := user.QuotaRule.Limit

	user.QuotaRule.Limit = limit
	user.QuotaRule.Edit(db.DefaultContext, &limit, nil)

	return func() {
		user.QuotaRule.Limit = previousLimit
		user.QuotaRule.Edit(db.DefaultContext, &previousLimit, nil)
	}
}

func (user *quotaWebEnvUser) ID() convertAs {
	return convertAs{
		asString: fmt.Sprintf("%d", user.User.ID),
	}
}

func (org *quotaWebEnvOrg) ID() convertAs {
	return convertAs{
		asString: fmt.Sprintf("%d", org.Org.ID),
	}
}

type convertAs struct {
	asString string
}

func (cas convertAs) AsString() string {
	return cas.asString
}

func (env *quotaWebEnv) Cleanup() {
	for i := len(env.cleaners) - 1; i >= 0; i-- {
		env.cleaners[i]()
	}
}

func (env *quotaWebEnv) As(t *testing.T, user quotaWebEnvUser) *quotaWebEnvAsContext {
	t.Helper()

	ctx := quotaWebEnvAsContext{
		t:    t,
		Doer: &user,
		Repo: user.Repo,

		Payload: Payload{},
	}
	return &ctx
}

func (env *quotaWebEnv) RunVisitAndPostToPageTests(t *testing.T, page string, payload *Payload, successStatus int) {
	t.Helper()

	// Visiting the page is always fine.
	env.As(t, env.Users.Limited).
		VisitPage(page).
		ExpectStatus(http.StatusOK)

	// Posting as the Limited user fails, because it is over quota.
	env.As(t, env.Users.Limited).
		With(Context{Payload: payload}).
		With(Context{
			Payload: &Payload{
				"uid": env.Users.Limited.ID().AsString(),
			},
		}).
		PostToPage(page).
		ExpectStatus(http.StatusRequestEntityTooLarge)

	// Migrating to a limited org also fails, for the same reason.
	env.As(t, env.Users.Limited).
		With(Context{Payload: payload}).
		With(Context{
			Payload: &Payload{
				"uid": env.Orgs.Limited.ID().AsString(),
			},
		}).
		PostToPage(page).
		ExpectStatus(http.StatusRequestEntityTooLarge)

	// Migrating to an unlimited repo works, however.
	env.As(t, env.Users.Limited).
		With(Context{Payload: payload}).
		With(Context{
			Payload: &Payload{
				"uid": env.Orgs.Unlimited.ID().AsString(),
			},
		}).
		PostToPage(page).
		ExpectStatus(successStatus)
}

func createQuotaWebEnv(t *testing.T) *quotaWebEnv {
	t.Helper()

	// *** helpers ***

	// Create a user, its quota group & rule
	makeUser := func(t *testing.T, limit int64) quotaWebEnvUser {
		t.Helper()

		user := quotaWebEnvUser{}

		// Create the user
		userName := gouuid.NewString()
		apiCreateUser(t, userName)
		user.User = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: userName})
		user.Session = loginUser(t, userName)

		// Create a repository for the user
		repo, _, _ := CreateDeclarativeRepoWithOptions(t, user.User, DeclarativeRepoOptions{})
		user.Repo = repo

		// Create a quota group for them
		group, err := quota_model.CreateGroup(db.DefaultContext, userName)
		assert.NoError(t, err)
		user.QuotaGroup = group

		// Create a rule
		rule, err := quota_model.CreateRule(db.DefaultContext, userName, limit, quota_model.LimitSubjects{quota_model.LimitSubjectSizeAll})
		assert.NoError(t, err)
		user.QuotaRule = rule

		// Add the rule to the group
		err = group.AddRuleByName(db.DefaultContext, rule.Name)
		assert.NoError(t, err)

		// Add the user to the group
		err = group.AddUserByID(db.DefaultContext, user.User.ID)
		assert.NoError(t, err)

		return user
	}

	// Create a user, its quota group & rule
	makeOrg := func(t *testing.T, owner *user_model.User, limit int64) quotaWebEnvOrg {
		t.Helper()

		org := quotaWebEnvOrg{}

		// Create the org
		userName := gouuid.NewString()
		org.Org = &org_model.Organization{
			Name: userName,
		}
		err := org_model.CreateOrganization(db.DefaultContext, org.Org, owner)
		assert.NoError(t, err)

		// Create a repository for the org
		orgUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: org.Org.ID})
		repo, _, _ := CreateDeclarativeRepoWithOptions(t, orgUser, DeclarativeRepoOptions{})
		org.Repo = repo

		// Create a quota group for them
		group, err := quota_model.CreateGroup(db.DefaultContext, userName)
		assert.NoError(t, err)
		org.QuotaGroup = group

		// Create a rule
		rule, err := quota_model.CreateRule(db.DefaultContext, userName, limit, quota_model.LimitSubjects{quota_model.LimitSubjectSizeAll})
		assert.NoError(t, err)
		org.QuotaRule = rule

		// Add the rule to the group
		err = group.AddRuleByName(db.DefaultContext, rule.Name)
		assert.NoError(t, err)

		// Add the org to the group
		err = group.AddUserByID(db.DefaultContext, org.Org.ID)
		assert.NoError(t, err)

		return org
	}

	env := quotaWebEnv{}
	env.cleaners = []func(){
		test.MockVariableValue(&setting.Quota.Enabled, true),
		test.MockVariableValue(&testWebRoutes, routers.NormalRoutes()),
	}

	// Create the limited user, and the various orgs
	env.Users.Limited = makeUser(t, int64(0))
	env.Orgs.Limited = makeOrg(t, env.Users.Limited.User, int64(0))
	env.Orgs.Unlimited = makeOrg(t, env.Users.Limited.User, int64(-1))

	return &env
}

// I am glad you read this far, but you now feel a pair of eyes watching you.
// Told you so.
