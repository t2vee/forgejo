// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/hostmatcher"
	"code.gitea.io/gitea/modules/setting"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookProxy(t *testing.T) {
	oldWebhook := setting.Webhook
	oldHTTPProxy := os.Getenv("http_proxy")
	oldHTTPSProxy := os.Getenv("https_proxy")
	t.Cleanup(func() {
		setting.Webhook = oldWebhook
		os.Setenv("http_proxy", oldHTTPProxy)
		os.Setenv("https_proxy", oldHTTPSProxy)
	})
	os.Unsetenv("http_proxy")
	os.Unsetenv("https_proxy")

	setting.Webhook.ProxyURL = "http://localhost:8080"
	setting.Webhook.ProxyURLFixed, _ = url.Parse(setting.Webhook.ProxyURL)
	setting.Webhook.ProxyHosts = []string{"*.discordapp.com", "discordapp.com"}

	allowedHostMatcher := hostmatcher.ParseHostMatchList("webhook.ALLOWED_HOST_LIST", "discordapp.com,s.discordapp.com")

	tests := []struct {
		req     string
		want    string
		wantErr bool
	}{
		{
			req:     "https://discordapp.com/api/webhooks/xxxxxxxxx/xxxxxxxxxxxxxxxxxxx",
			want:    "http://localhost:8080",
			wantErr: false,
		},
		{
			req:     "http://s.discordapp.com/assets/xxxxxx",
			want:    "http://localhost:8080",
			wantErr: false,
		},
		{
			req:     "http://github.com/a/b",
			want:    "",
			wantErr: false,
		},
		{
			req:     "http://www.discordapp.com/assets/xxxxxx",
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.req, func(t *testing.T) {
			req, err := http.NewRequest("POST", tt.req, nil)
			require.NoError(t, err)

			u, err := webhookProxy(allowedHostMatcher)(req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			got := ""
			if u != nil {
				got = u.String()
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWebhookDeliverAuthorizationHeader(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	done := make(chan struct{}, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/webhook", r.URL.Path)
		assert.Equal(t, "Bearer s3cr3t-t0ken", r.Header.Get("Authorization"))
		w.WriteHeader(200)
		done <- struct{}{}
	}))
	t.Cleanup(s.Close)

	hook := &webhook_model.Webhook{
		RepoID:      3,
		URL:         s.URL + "/webhook",
		ContentType: webhook_model.ContentTypeJSON,
		IsActive:    true,
		Type:        webhook_module.GITEA,
	}
	err := hook.SetHeaderAuthorization("Bearer s3cr3t-t0ken")
	assert.NoError(t, err)
	assert.NoError(t, webhook_model.CreateWebhook(db.DefaultContext, hook))
	db.GetEngine(db.DefaultContext).NoAutoTime().DB().Logger.ShowSQL(true)

	hookTask := &webhook_model.HookTask{
		HookID:            hook.ID,
		EventType:         webhook_module.HookEventPush,
		RequestMethod:     "GET",
		RequestURL:        s.URL + "/webhook",
		AddDefaultHeaders: true,
	}

	hookTask, err = webhook_model.CreateHookTask(db.DefaultContext, hookTask)
	assert.NoError(t, err)
	if !assert.NotNil(t, hookTask) {
		return
	}

	assert.NoError(t, Deliver(context.Background(), hookTask))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("waited to long for request to happen")
	}

	assert.True(t, hookTask.IsSucceed)
}

func TestWebhookDeliverHookTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	done := make(chan struct{}, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/webhook", r.URL.Path)
		assert.Equal(t, "Bearer s3cr3t-t0ken", r.Header.Get("Authorization"))
		assert.Equal(t, "push", r.Header.Get("X-GitHub-Event"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, `{"data": 42}`, string(body))
		w.WriteHeader(200)
		done <- struct{}{}
	}))
	t.Cleanup(s.Close)

	hook := &webhook_model.Webhook{
		RepoID:   3,
		IsActive: true,
		Type:     webhook_module.GITEA,
	}
	err := hook.SetHeaderAuthorization("Bearer s3cr3t-t0ken")
	assert.NoError(t, err)
	assert.NoError(t, webhook_model.CreateWebhook(db.DefaultContext, hook))
	db.GetEngine(db.DefaultContext).NoAutoTime().DB().Logger.ShowSQL(true)

	hookTask := &webhook_model.HookTask{
		HookID:            hook.ID,
		EventType:         webhook_module.HookEventPush,
		RequestMethod:     "PUT",
		RequestURL:        s.URL + "/webhook",
		RequestHeader:     `Content-Type: application/json`,
		PayloadContent:    `{"data": 42}`,
		AddDefaultHeaders: true,
	}

	hookTask, err = webhook_model.CreateHookTask(db.DefaultContext, hookTask)
	assert.NoError(t, err)
	if !assert.NotNil(t, hookTask) {
		return
	}

	assert.NoError(t, Deliver(context.Background(), hookTask))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("waited to long for request to happen")
	}

	assert.True(t, hookTask.IsSucceed)
}
