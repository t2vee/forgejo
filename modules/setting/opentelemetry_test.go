// // Copyright 2024 The Forgejo Authors. All rights reserved.
// // SPDX-License-Identifier: MIT

package setting

import (
	"testing"
	"time"

	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestOpenTelemetryConfiguration(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	iniStr := ``
	cfg, err := NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.Nil(t, OpenTelemetry.Traces.Endpoint)
	assert.False(t, IsOpenTelemetryEnabled())

	iniStr = `
	[opentelemetry]
	ENDPOINT = http://jaeger:4317/

	TIMEOUT = 30s
	SAMPLER = always_on
	COMPRESSION = gzip
	INSECURE = TRUE
	HEADERS=foo=bar,overwrite=false
	[opentelemetry.resources]
	SERVICE_NAME = test service
	RESOURCE_ATTRIBUTES = foo=bar

	[opentelemetry.traces]
	TIMEOUT=5s
	HEADERS=overwrite=true,foobar=barfoo
	`
	cfg, err = NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)

	assert.True(t, IsOpenTelemetryEnabled())
	assert.Equal(t, "test service", OpenTelemetry.Resource.ServiceName)
	assert.Equal(t, "foo=bar", OpenTelemetry.Resource.Attributes)
	assert.Equal(t, 5*time.Second, OpenTelemetry.Traces.Timeout)
	assert.Equal(t, "gzip", OpenTelemetry.Traces.Compression)
	assert.Equal(t, "always_on", OpenTelemetry.Traces.Sampler)
	assert.Equal(t, "http://jaeger:4317/", OpenTelemetry.Traces.Endpoint.String())
	assert.Contains(t, OpenTelemetry.Traces.Headers, "foo")
	assert.Equal(t, OpenTelemetry.Traces.Headers["foo"], "bar")
	assert.Contains(t, OpenTelemetry.Traces.Headers, "foobar")
	assert.Equal(t, OpenTelemetry.Traces.Headers["foobar"], "barfoo")
	assert.Contains(t, OpenTelemetry.Traces.Headers, "overwrite")
	assert.Equal(t, OpenTelemetry.Traces.Headers["overwrite"], "true")
}

func TestOpentelemetryBadConfigs(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	iniStr := `
	[opentelemetry]
	ENDPOINT = jaeger:4317/
	`
	cfg, err := NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)

	assert.False(t, IsOpenTelemetryEnabled())
	assert.Nil(t, OpenTelemetry.Traces.Endpoint)

	iniStr = ``
	cfg, err = NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.False(t, IsOpenTelemetryEnabled())

	iniStr = `
	[opentelemetry]
	ENDPOINT = http://jaeger:4317/

	TIMEOUT = abc
	SAMPLER = not existing one
	COMPRESSION = foo
	HEADERS=%s=bar,foo=%h,foo
	[opentelemetry.resources]
	SERVICE_NAME =
	`
	cfg, err = NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.True(t, IsOpenTelemetryEnabled())
	assert.Equal(t, "forgejo", OpenTelemetry.Resource.ServiceName)
	assert.Equal(t, 10*time.Second, OpenTelemetry.Traces.Timeout)
	assert.Equal(t, "parentbased_always_on", OpenTelemetry.Traces.Sampler)
	assert.Equal(t, "http://jaeger:4317/", OpenTelemetry.Traces.Endpoint.String())
	assert.Empty(t, OpenTelemetry.Traces.Headers)
}
