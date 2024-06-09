package opentelemetry

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/setting"

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

func TestFailExporter(t *testing.T) {
	setting.OpenTelemetry.Traces.Endpoint = ":4317"
	ctx := context.Background()
	shutdown, err := SetupOTel(ctx)
	assert.Error(t, err)
	defer shutdown(ctx)
	tracer := otel.Tracer("test_fail")

	_, span := tracer.Start(ctx, "test span")

	assert.False(t, span.SpanContext().HasTraceID())
	assert.False(t, span.SpanContext().HasSpanID())
}

func TestOtelIntegration(t *testing.T) {
	if os.Getenv("TEST_OTEL_COLLECTOR") == "" {
		t.Skip("Jaeger not set, skipping otel integration test")
	}

	setting.OpenTelemetry.Traces.Endpoint = os.Getenv("TEST_OTEL_COLLECTOR")
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
	resp, err := http.Get("http://jaeger:16686/api/services")

	assert.NoError(t, err)

	apiResponse, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	strings.Contains(string(apiResponse), "forgejo")
}

func TestExporter(t *testing.T) {
	setting.OpenTelemetry.Traces.Endpoint = "http://localhost:4317"
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
