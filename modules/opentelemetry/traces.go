package opentelemetry

import (
	"context"
	"crypto/tls"

	"code.gitea.io/gitea/modules/setting"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
)

var newTraceExporter = func(ctx context.Context) (sdktrace.SpanExporter, error) {
	endpoint := setting.OpenTelemetry.Traces.Endpoint

	opts := []otlptracegrpc.Option{}

	tlsConf := &tls.Config{}
	opts = append(opts, otlptracegrpc.WithEndpoint(endpoint.Host))
	opts = append(opts, otlptracegrpc.WithTimeout(setting.OpenTelemetry.Traces.Timeout))
	if setting.OpenTelemetry.Traces.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if setting.OpenTelemetry.Traces.Compression != "" {
		opts = append(opts, otlptracegrpc.WithCompressor(setting.OpenTelemetry.Traces.Compression))
	}
	withCertPool(setting.OpenTelemetry.Traces.Certificate, tlsConf)
	withClientCert(setting.OpenTelemetry.Traces.ClientCertificate, setting.OpenTelemetry.Traces.ClientKey, tlsConf)
	if tlsConf.RootCAs != nil || len(tlsConf.Certificates) > 0 {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(
			credentials.NewTLS(tlsConf),
		))
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

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(setting.OpenTelemetry.Traces.Sampler),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(r),
	)
	otel.SetTracerProvider(traceProvider)
	return traceProvider.Shutdown, nil
}
