// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"os"
	"testing"

	"code.gitea.io/gitea/cmd"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/migrations"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	f3_forges "lab.forgefriends.org/friendlyforgeformat/gof3/forges"
	f3_util "lab.forgefriends.org/friendlyforgeformat/gof3/util"
)

func Test_CmdF3(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		AllowLocalNetworks := setting.Migrations.AllowLocalNetworks
		setting.F3.Enabled = true
		setting.Migrations.AllowLocalNetworks = true
		// without migrations.Init() AllowLocalNetworks = true is not effective and
		// a http call fails with "...migration can only call allowed HTTP servers..."
		migrations.Init()
		AppVer := setting.AppVer
		// Gitea SDK (go-sdk) need to parse the AppVer from server response, so we must set it to a valid version string.
		setting.AppVer = "1.16.0"
		defer func() {
			setting.Migrations.AllowLocalNetworks = AllowLocalNetworks
			setting.AppVer = AppVer
		}()

		//
		// Step 1: create a fixture
		//
		fixture := f3_forges.NewFixture(t, f3_forges.FixtureF3Factory)
		fixture.NewUser(1234)
		fixture.NewMilestone()
		fixture.NewLabel()
		fixture.NewIssue()
		fixture.NewTopic()
		fixture.NewRepository()
		fixture.NewRelease()
		fixture.NewAsset()
		fixture.NewIssueComment(nil)
		fixture.NewIssueReaction()

		//
		// Step 2: import the fixture into Gitea
		//
		cmd.CmdF3.Action = func(ctx *cli.Context) error { return cmd.RunF3(context.Background(), ctx) }
		{
			realStdout := os.Stdout // Backup Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			app := cli.NewApp()
			app.Writer = w
			app.ErrWriter = w
			app.Commands = []*cli.Command{cmd.CmdF3}
			assert.NoError(t, app.Run([]string{"forgejo", "f3", "--import", "--directory", fixture.ForgeRoot.GetDirectory()}))

			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			commandOutput := buf.String()
			assert.EqualValues(t, "imported\n", commandOutput)
			os.Stdout = realStdout
		}

		//
		// Step 3: export Gitea into F3
		//
		directory := t.TempDir()
		{
			realStdout := os.Stdout // Backup Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			app := cli.NewApp()
			app.Writer = w
			app.ErrWriter = w
			app.Commands = []*cli.Command{cmd.CmdF3}
			assert.NoError(t, app.Run([]string{"forgejo", "f3", "--export", "--no-pull-request", "--user", fixture.UserFormat.UserName, "--repository", fixture.ProjectFormat.Name, "--directory", directory}))

			w.Close()
			var buf bytes.Buffer
			io.Copy(&buf, r)
			commandOutput := buf.String()
			assert.EqualValues(t, "exported\n", commandOutput)
			os.Stdout = realStdout

		}

		//
		// Step 4: verify the export and import are equivalent
		//
		files := f3_util.Command(context.Background(), "find", directory)
		assert.Contains(t, files, "/label/")
		assert.Contains(t, files, "/issue/")
		assert.Contains(t, files, "/milestone/")
		assert.Contains(t, files, "/topic/")
		assert.Contains(t, files, "/release/")
		assert.Contains(t, files, "/asset/")
		assert.Contains(t, files, "/reaction/")
	})
}
