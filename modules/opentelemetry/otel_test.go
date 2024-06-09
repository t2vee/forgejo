package opentelemetry

import (
	"context"
	"testing"

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
