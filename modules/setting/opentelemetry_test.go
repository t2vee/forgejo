// // Copyright 2024 The Forgejo Authors. All rights reserved.
// // SPDX-License-Identifier: MIT

package setting

import (
	"testing"
	"time"

	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
	COMPRESSION = gzip
	INSECURE = TRUE
	HEADERS=foo=bar,overwrite=false
	[opentelemetry.resources]
	SERVICE_NAME = test service
	RESOURCE_ATTRIBUTES = foo=bar
	[opentelemetry.traces]
	SAMPLER = always_on
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
	assert.Equal(t, sdktrace.AlwaysSample(), OpenTelemetry.Traces.Sampler)
	assert.Equal(t, "http://jaeger:4317/", OpenTelemetry.Traces.Endpoint.String())
	assert.Contains(t, OpenTelemetry.Traces.Headers, "foo")
	assert.Equal(t, OpenTelemetry.Traces.Headers["foo"], "bar")
	assert.Contains(t, OpenTelemetry.Traces.Headers, "foobar")
	assert.Equal(t, OpenTelemetry.Traces.Headers["foobar"], "barfoo")
	assert.Contains(t, OpenTelemetry.Traces.Headers, "overwrite")
	assert.Equal(t, OpenTelemetry.Traces.Headers["overwrite"], "true")
}

func TestSamplerCombinations(t *testing.T) {
	defer test.MockProtect(&OpenTelemetry)()
	type config struct {
		IniCfg   string
		Expected sdktrace.Sampler
	}
	testSamplers := []config{
		{`[opentelemetry]
	ENDPOINT=http://localhost:4317
	[opentelemetry.traces]
  SAMPLER = always_on
  SAMPLER_ARG = nothing`, sdktrace.AlwaysSample()},
		{`[opentelemetry]
	ENDPOINT=http://localhost:4317
	[opentelemetry.traces]
  SAMPLER = always_off`, sdktrace.NeverSample()},
		{`[opentelemetry]
	ENDPOINT=http://localhost:4317
	[opentelemetry.traces]
  SAMPLER = traceidratio
  SAMPLER_ARG = 0.7`, sdktrace.TraceIDRatioBased(0.7)},
		{`[opentelemetry]
	ENDPOINT=http://localhost:4317
	[opentelemetry.traces]
  SAMPLER = traceidratio
  SAMPLER_ARG = badarg`, sdktrace.TraceIDRatioBased(1)},
		{`[opentelemetry]
	ENDPOINT=http://localhost:4317
	[opentelemetry.traces]
  SAMPLER = parentbased_always_off`, sdktrace.ParentBased(sdktrace.NeverSample())},
		{`[opentelemetry]
	ENDPOINT=http://localhost:4317
	[opentelemetry.traces]
  SAMPLER = parentbased_always_of`, sdktrace.ParentBased(sdktrace.AlwaysSample())},
		{`[opentelemetry]
	ENDPOINT=http://localhost:4317
	[opentelemetry.traces]
  SAMPLER = parentbased_traceidratio
  SAMPLER_ARG = 0.3`, sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.3))},
		{`[opentelemetry]
	ENDPOINT=http://localhost:4317
	[opentelemetry.traces]
  SAMPLER = parentbased_traceidratio
  SAMPLER_ARG = badarg`, sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1))},
	}

	for _, sampler := range testSamplers {
		cfg, err := NewConfigProviderFromData(sampler.IniCfg)
		assert.NoError(t, err)
		loadOpenTelemetryFrom(cfg)
		assert.Equal(t, sampler.Expected, OpenTelemetry.Traces.Sampler)
	}
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
	COMPRESSION = foo
	HEADERS=%s=bar,foo=%h,foo
	[opentelemetry.resources]
	SERVICE_NAME =
	[opentelemetry.traces]
  SAMPLER = not existing one
	`
	cfg, err = NewConfigProviderFromData(iniStr)
	assert.NoError(t, err)
	loadOpenTelemetryFrom(cfg)
	assert.True(t, IsOpenTelemetryEnabled())
	assert.Equal(t, "forgejo", OpenTelemetry.Resource.ServiceName)
	assert.Equal(t, 10*time.Second, OpenTelemetry.Traces.Timeout)
	assert.Equal(t, sdktrace.ParentBased(sdktrace.AlwaysSample()), OpenTelemetry.Traces.Sampler)
	assert.Equal(t, "http://jaeger:4317/", OpenTelemetry.Traces.Endpoint.String())
	assert.Empty(t, OpenTelemetry.Traces.Headers)
}
