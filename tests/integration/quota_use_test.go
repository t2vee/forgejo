// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2
//
// Hello! Stare at this code long enough, and it might stare back.

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	gouuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWebQuotaEnforcementRepoMigrate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		// Visiting the repo migrate page is always allowed, because we can migrate
		// *into* another org, for example.
		env.As(t, env.Users.Limited).
			VisitPage("/repo/migrate").
			ExpectStatus(http.StatusOK)

		// Migrating to our user fails, because we're over quota.
		env.As(t, env.Users.Limited).
			With(Context{
				Payload: &Payload{
					"uid":        env.Users.Limited.ID().AsString(),
					"repo_name":  "failing-migration",
					"clone_addr": env.Users.Limited.Repo.HTMLURL() + ".git",
				},
			}).
			PostToPage("/repo/migrate").
			ExpectStatus(http.StatusRequestEntityTooLarge)

		// Migrating to a limited org also fails, for the same reason.
		env.As(t, env.Users.Limited).
			With(Context{
				Payload: &Payload{
					"uid":        env.Orgs.Limited.ID().AsString(),
					"repo_name":  "failing-migration",
					"clone_addr": env.Users.Limited.Repo.HTMLURL() + ".git",
				},
			}).
			PostToPage("/repo/migrate").
			ExpectStatus(http.StatusRequestEntityTooLarge)

		// Migrating to an unlimited repo works, however.
		env.As(t, env.Users.Limited).
			With(Context{
				Payload: &Payload{
					"uid":        env.Orgs.Unlimited.ID().AsString(),
					"repo_name":  "failing-migration",
					"clone_addr": env.Users.Limited.Repo.HTMLURL() + ".git",
				},
			}).
			PostToPage("/repo/migrate").
			ExpectStatus(http.StatusOK)
	})
}

func TestWebQuotaEnforcementRepoCreate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		env := createQuotaWebEnv(t)
		defer env.Cleanup()

		// Visiting the repo create page is always allowed, because we can
		// create *into* another org, for example.
		env.As(t, env.Users.Limited).
			VisitPage("/repo/create").
			ExpectStatus(http.StatusOK)

		// Creating into *our* repo fails.
		env.As(t, env.Users.Limited).
			With(Context{
				Payload: &Payload{
					"uid": env.Users.Limited.ID().AsString(),
				},
			}).
			PostToPage("/repo/create").
			ExpectStatus(http.StatusRequestEntityTooLarge)

		// Creating into a limited org also fails.
		env.As(t, env.Users.Limited).
			With(Context{
				Payload: &Payload{
					"uid": env.Orgs.Limited.ID().AsString(),
				},
			}).
			PostToPage("/repo/create").
			ExpectStatus(http.StatusRequestEntityTooLarge)

		// Creating into an unlimited org works.
		env.As(t, env.Users.Limited).
			With(Context{
				Payload: &Payload{
					"uid": env.Orgs.Unlimited.ID().AsString(),
				},
			}).
			PostToPage("/repo/create").
			ExpectStatus(http.StatusOK)
	})
}

/*
Routes:
  org.PackagesRuleAdd{,Post}
  org.PackagesRuleEdit{,Post}

  org.InitializeCargoIndex
  org.RebuildCargoIndex

  PR repo.Create => should filter out invalid targets
  DONE repo.CreatePost

  PR repo.Migrate => should filter out invalid targets
  DONE repo.MigratePost

  repo.ForkByID => should filter out invalid targets

  user.PackageSettingsPost => verify target? if this is where assignment happens

  protected tags? do they ++ git size, or are they db only like protected branch settings?

  repo_lfs.LFSAutoAssociate? do we care?

  repo.MigrateRetryPost -> needs quota, probably
  repo.MigrateCancelPost -> probs doesn't need quota

  repo.ActionTransfer() => quota!

  repo.CompareAndPullRequest{,Post} => needs to check target quota, I think

  repo.Fork{,Post} => filter & target check

  repo.NewIssuePost => check for attachments?

  repo.UpdateIssueContent => check if this deals with attachments.

  repo.NewComment => check for attachments

  repo.UploadIssueAttachment => route check

  repo.UpdateCommentContent => check if it deals w/ attachments

  repo.DiffPreviewPost => need quota here?

  repo.EditFile, repo.NewFile, repo.NewFilePost, repo.UploadFilePost => route check

  repo.NewDiffPatch, repo.CherryPick => wtf do these do?

  repo.UploadFileToServer => where does this upload it to? quota check needed, but what subject?

  /branches/_new => always quota check
  incl: repo.CreateBranch

  repo.RestoreBranch{,Post} => route check

  repo.NewRelease{,Post} => route check? since it can create tags.
  repo.EditRelease{,Post} => same

  repo.UploadReleaseAttachment => route check

  repo.{Disable,Enable}Workflowfile => does this touch the git repo or anything?

  repo.WikiPost => route check

  repo.MergePullRequest => route check

  repo.UpdatePullRequest => no check needed

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

	request *RequestWrapper
}

type Context struct {
	Repo    *repo_model.Repository
	Payload *Payload
}

func (ctx *quotaWebEnvAsContext) With(opts Context) *quotaWebEnvAsContext {
	if opts.Repo != nil {
		ctx.Repo = opts.Repo
	}
	if opts.Payload != nil {
		ctx.Payload = *opts.Payload
	}
	return ctx
}

func (ctx *quotaWebEnvAsContext) VisitPage(page string) *quotaWebEnvAsContext {
	ctx.t.Helper()

	ctx.request = NewRequest(ctx.t, "GET", page)

	return ctx
}

func (ctx *quotaWebEnvAsContext) ExpectStatus(status int) *httptest.ResponseRecorder {
	ctx.t.Helper()

	return ctx.Doer.Session.MakeRequest(ctx.t, ctx.request, status)
}

func (ctx *quotaWebEnvAsContext) VisitRepoPage(page string) *quotaWebEnvAsContext {
	ctx.t.Helper()

	return ctx.VisitPage(ctx.Repo.HTMLURL() + page)
}

func (ctx *quotaWebEnvAsContext) PostToPage(page string) *quotaWebEnvAsContext {
	ctx.t.Helper()

	payload := ctx.Payload
	payload["_csrf"] = GetCSRF(ctx.t, ctx.Doer.Session, page)

	ctx.request = NewRequestWithValues(ctx.t, "POST", page, payload)

	return ctx
}

func (ctx *quotaWebEnvAsContext) PostToRepoPage(page string) *quotaWebEnvAsContext {
	ctx.t.Helper()

	return ctx.PostToPage(ctx.Repo.HTMLURL() + page)
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

func (self convertAs) AsString() string {
	return self.asString
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
	}
	return &ctx
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

		// Create the user
		userName := gouuid.NewString()
		org.Org = &org_model.Organization{
			Name: userName,
		}
		err := org_model.CreateOrganization(db.DefaultContext, org.Org, owner)
		assert.NoError(t, err)

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
