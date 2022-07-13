// Copyright 2022 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package activitypub

import (
	"context"
	"fmt"
	"strings"

	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	pull_service "code.gitea.io/gitea/services/pull"

	ap "github.com/go-ap/activitypub"
)

func PullRequest(ctx context.Context, activity ap.Move) {
	actorIRI := activity.AttributedTo.GetLink()
	actorIRISplit := strings.Split(actorIRI.String(), "/")
	actorName := actorIRISplit[len(actorIRISplit)-1] + "@" + actorIRISplit[2]
	err := FederatedUserNew(actorName, actorIRI)
	if err != nil {
		log.Warn("Couldn't create new user", err)
	}
	actorUser, err := user_model.GetUserByName(ctx, actorName)
	if err != nil {
		log.Warn("Couldn't find actor", err)
	}

	// This code is really messy
	// The IRI processing stuff should be in a separate function
	originIRI := activity.Origin.GetLink()
	originIRISplit := strings.Split(originIRI.String(), "/")
	originInstance := originIRISplit[2]
	originUsername := originIRISplit[3]
	originReponame := originIRISplit[4]
	originBranch := originIRISplit[len(originIRISplit)-1]
	originRepo, _ := repo_model.GetRepositoryByOwnerAndName(originUsername+"@"+originInstance, originReponame)

	targetIRI := activity.Target.GetLink()
	targetIRISplit := strings.Split(targetIRI.String(), "/")
	// targetInstance := targetIRISplit[2]
	targetUsername := targetIRISplit[3]
	targetReponame := targetIRISplit[4]
	targetBranch := targetIRISplit[len(targetIRISplit)-1]

	targetRepo, _ := repo_model.GetRepositoryByOwnerAndName(targetUsername, targetReponame)

	prIssue := &issues_model.Issue{
		RepoID:   targetRepo.ID,
		Title:    "Hello from test.exozy.me!", // Don't hardcode, get the title from the Ticket object
		PosterID: actorUser.ID,
		Poster:   actorUser,
		IsPull:   true,
		Content:  "🎉",
	}

	pr := &issues_model.PullRequest{
		HeadRepoID:   originRepo.ID,
		BaseRepoID:   targetRepo.ID,
		HeadBranch:   originBranch,
		HeadCommitID: "73f228996f27fad2c7bb60435f912d943b66b0ee", // hardcoded for now
		BaseBranch:   targetBranch,
		HeadRepo:     originRepo,
		BaseRepo:     targetRepo,
		MergeBase:    "",
		Type:         issues_model.PullRequestGitea,
	}

	err = pull_service.NewPullRequest(ctx, targetRepo, prIssue, []int64{}, []string{}, pr, []int64{})
	fmt.Println(err)
}
