// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
)

var testSamplers = []string{
	AlwaysOff, AlwaysOn, ParentBasedAlwaysOff, ParentBasedAlwaysOn,
}

func TestNoopDefault(t *testing.T) {
	ctx := context.Background()
	shutdown, err := SetupOTel(ctx)
	assert.NoError(t, err)
	defer shutdown(ctx)
	tracer := otel.Tracer("test_noop")

	_, span := tracer.Start(ctx, "test span")

	assert.False(t, span.SpanContext().HasTraceID())
	assert.False(t, span.SpanContext().HasSpanID())
}

func TestOtelIntegration(t *testing.T) {
	if os.Getenv("TEST_OTEL_COLLECTOR") == "" {
		t.Skip("Jaeger not set, skipping otel integration test")
	}

	jaeger, err := url.Parse(os.Getenv("TEST_OTEL_COLLECTOR"))
	assert.NoError(t, err)

	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.ServiceName, "forgejo-integration")()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Endpoint, jaeger)()
	ctx := context.Background()
	shutdown, err := SetupOTel(ctx)
	assert.NoError(t, err)
	defer shutdown(ctx)
	tracer := otel.Tracer("test_jaeger")
	_, span := tracer.Start(ctx, "test span")
	assert.True(t, span.SpanContext().HasTraceID())
	assert.True(t, span.SpanContext().HasSpanID())

	span.End()
	// Give the exporter time to send the span
	time.Sleep(10 * time.Second)
	resp, err := http.Get("http://localhost:16686/api/services")

	assert.NoError(t, err)

	apiResponse, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	strings.Contains(string(apiResponse), setting.OpenTelemetry.Resource.ServiceName)
}

func TestExporter(t *testing.T) {

	endpoint, err := url.Parse("http://localhost:4317")
	assert.NoError(t, err)

	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Endpoint, endpoint)()
	defer test.MockProtect[string](&setting.OpenTelemetry.Traces.Sampler)()
	for _, sampler := range testSamplers {
		setting.OpenTelemetry.Traces.Sampler = sampler
		ctx := context.Background()
		shutdown, err := SetupOTel(ctx)
		assert.NoError(t, err)
		defer shutdown(ctx)
		tracer := otel.Tracer("test_grpc")

		_, span := tracer.Start(ctx, "test span")

		assert.True(t, span.SpanContext().HasTraceID())
		assert.True(t, span.SpanContext().HasSpanID())
	}
}
