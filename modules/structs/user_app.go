// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package structs

import (
	"encoding/base64"
)

// BasicAuthEncode generate base64 of basic auth head
func BasicAuthEncode(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

// AccessToken represents an API access token.
// swagger:response AccessToken
type AccessToken struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Token          string `json:"sha1"`
	TokenLastEight string `json:"token_last_eight"`
}

// AccessTokenList represents a list of API access token.
// swagger:response AccessTokenList
type AccessTokenList []*AccessToken

// CreateAccessTokenOption options when create access token
// swagger:parameters userCreateToken
type CreateAccessTokenOption struct {
	Name string `json:"name" binding:"Required"`
}

// CreateOAuth2ApplicationOptions holds options to create an oauth2 application
// swagger:parameters userCreateOAuth2Application
type CreateOAuth2ApplicationOptions struct {
	Name         string   `json:"name" binding:"Required"`
	RedirectURIs []string `json:"redirect-uris" binding:"Required"`
}

// OAuth2Application represents an OAuth2 application.
// swagger:response OAuth2Application
type OAuth2Application struct {
	Name         string   `json:"name"`
	ClientID     string   `json:"client-id"`
	ClientSecret string   `json:"client-secret"`
	RedirectURIs []string `json:"redirect-uris"`
}
