// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// env keys for git hooks need
const (
	EnvRepoName     = "GITEA_REPO_NAME"
	EnvRepoUsername = "GITEA_REPO_USER_NAME"
	EnvRepoIsWiki   = "GITEA_REPO_IS_WIKI"
	EnvPusherName   = "GITEA_PUSHER_NAME"
	EnvPusherEmail  = "GITEA_PUSHER_EMAIL"
	EnvPusherID     = "GITEA_PUSHER_ID"
	EnvKeyID        = "GITEA_KEY_ID"
	EnvIsDeployKey  = "GITEA_IS_DEPLOY_KEY"
	EnvIsInternal   = "GITEA_INTERNAL_PUSH"
)

// InternalPushingEnvironment returns an os environment to switch off hooks on push
// It is recommended to avoid using this unless you are pushing within a transaction
// or if you absolutely are sure that post-receive and pre-receive will do nothing
// We provide the full pushing-environment for other hook providers
func InternalPushingEnvironment(doer *User, repo *Repository) []string {
	fmt.Printf("%v InternalPushingEnvironment\n", time.Now().Format("15:04:05.000000"))
	return append(PushingEnvironment(doer, repo),
		EnvIsInternal+"=true",
	)
}

// PushingEnvironment returns an os environment to allow hooks to work on push
func PushingEnvironment(doer *User, repo *Repository) []string {
	return FullPushingEnvironment(doer, doer, repo, repo.Name, 0)
}

// FullPushingEnvironment returns an os environment to allow hooks to work on push
func FullPushingEnvironment(author, committer *User, repo *Repository, repoName string, prID int64) []string {
	start := time.Now()
	fmt.Printf("%v FullPushingEnvironment\n", time.Now().Format("15:04:05.000000"))
	isWiki := "false"
	if strings.HasSuffix(repoName, ".wiki") {
		isWiki = "true"
	}

	authorSig := author.NewGitSig()
	committerSig := committer.NewGitSig()

	// We should add "SSH_ORIGINAL_COMMAND=gitea-internal",
	// once we have hook and pushing infrastructure working correctly
	fmt.Printf("FullPushingEnvironment time taken: %v\n\n", time.Since(start))
	return append(os.Environ(),
		"GIT_AUTHOR_NAME="+authorSig.Name,
		"GIT_AUTHOR_EMAIL="+authorSig.Email,
		"GIT_COMMITTER_NAME="+committerSig.Name,
		"GIT_COMMITTER_EMAIL="+committerSig.Email,
		EnvRepoName+"="+repoName,
		EnvRepoUsername+"="+repo.OwnerName,
		EnvRepoIsWiki+"="+isWiki,
		EnvPusherName+"="+committer.Name,
		EnvPusherID+"="+fmt.Sprintf("%d", committer.ID),
		ProtectedBranchRepoID+"="+fmt.Sprintf("%d", repo.ID),
		ProtectedBranchPRID+"="+fmt.Sprintf("%d", prID),
		"SSH_ORIGINAL_COMMAND=gitea-internal",
		"GIT_TRACE=true",
		"GIT_TRACE_PACK_ACCESS=true",
		"GIT_TRACE_PERFORMANCE=true",
		"GIT_TRACE_SETUP=true",
	)

}
