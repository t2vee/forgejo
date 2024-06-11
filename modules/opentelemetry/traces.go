package opentelemetry

import (
	"context"

	"code.gitea.io/gitea/modules/setting"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var newTraceExporter = func(ctx context.Context) (sdktrace.SpanExporter, error) {
	endpoint := setting.OpenTelemetry.Traces.Endpoint

	opts := []otlptracegrpc.Option{}
	opts = append(opts, otlptracegrpc.WithEndpoint(endpoint.Host))
	opts = append(opts, otlptracegrpc.WithTimeout(setting.OpenTelemetry.Traces.Timeout))
	if setting.OpenTelemetry.Traces.Insecure || endpoint.Scheme == "http" || endpoint.Scheme == "unix" {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if setting.OpenTelemetry.Traces.Compression == Gzip {
		opts = append(opts, otlptracegrpc.WithCompressor(setting.OpenTelemetry.Traces.Compression))
	}

	return otlptracegrpc.New(ctx, opts...)
}

// Create new and register trace provider from user defined configuration
func setupTraceProvider(ctx context.Context, r *resource.Resource) (func(context.Context) error, error) {
	if setting.OpenTelemetry.Traces.Endpoint == nil {
		return func(ctx context.Context) error { return nil }, nil
	}
	traceExporter, err := newTraceExporter(ctx)
	if err != nil {
		return nil, err
	}

	sampler := newSampler()
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(r),
	)
	otel.SetTracerProvider(traceProvider)
	return traceProvider.Shutdown, nil
}
