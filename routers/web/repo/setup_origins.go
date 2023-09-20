package repo

import (
	"code.gitea.io/gitea/models"
	ctx "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/services/origins"
)

// TODO: hello frontender here you will can work on UI and forms retrieve

// SetupSourcesPost save a new source into database
func SetupSourcesPost(ctx *ctx.Context) {
	err := models.SaveOrigin(ctx, &models.Origin{
		UserID:         ctx.Doer.ID,
		Type:           models.GithubStarred,
		RemoteUsername: "cassiozareck",
		Token:          ""})

	if err != nil {
		log.Error("Couldn't save source into database: ", err)
	}
}

// SyncSourcesPost synchronizes the sources
func SyncSourcesPost(ctx *ctx.Context) {
	originSyncer := origins.NewOriginSyncer(ctx, ctx.Doer, ctx.Doer, 4)
	err := originSyncer.Fetch()
	if err != nil {
		log.Error("Couldn't fetch origin", err)
	}

	originSyncer.Sync()
}
