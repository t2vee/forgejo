// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"github.com/go-logr/logr/funcr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// type Compression string

const (
	None string = ""     // No compression
	Gzip string = "gzip" // Gzip compression
)

const (
	AlwaysOn                string = "always_on"
	AlwaysOff               string = "always_off"
	TraceIDRatio            string = "traceidratio"
	ParentBasedAlwaysOn     string = "parentbased_always_on"
	ParentBasedAlwaysOff    string = "parentbased_always_off"
	ParentBasedTraceIDRatio string = "parentbased_traceidratio"
)

func SetupOTel(ctx context.Context) (shutdown func(context.Context) error, err error) {
	// Redirect otel logger to write to common forgejo log at info
	logWrap := funcr.New(func(prefix, args string) {
		log.Info(fmt.Sprint(prefix, args))
	}, funcr.Options{})
	otel.SetLogger(logWrap)
	// Redirect error handling to forgejo log as well
	otel.SetErrorHandler(otelErrorHandler{})

	var shutdownFuncs []func(context.Context) error

	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	otel.SetTextMapPropagator(newPropagator())

	traceShutdown, err := setupTraceProvider(ctx)
	if err != nil {
		handleErr(err)
		return shutdown, err
	}

	shutdownFuncs = append(shutdownFuncs, traceShutdown)
	return shutdown, nil
}

func newTraceExporter(ctx context.Context) (*otlptrace.Exporter, error) {
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

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// Create new and register trace provider from user defined configuration
func setupTraceProvider(ctx context.Context) (func(context.Context) error, error) {
	if setting.OpenTelemetry.Traces.Endpoint == nil {
		return func(ctx context.Context) error { return nil }, nil
	}
	traceExporter, err := newTraceExporter(ctx)
	if err != nil {
		return nil, err
	}
	r, err := newResource(ctx)
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

func newSampler() sdktrace.Sampler {
	switch setting.OpenTelemetry.Traces.Sampler {
	case AlwaysOn:
		return sdktrace.AlwaysSample()
	case AlwaysOff:
		return sdktrace.NeverSample()
	case TraceIDRatio:
		ratio, err := strconv.ParseFloat(setting.OpenTelemetry.Traces.SamplerArg, 64)
		if err != nil {
			ratio = 1
		}
		return sdktrace.TraceIDRatioBased(ratio)
	case ParentBasedTraceIDRatio:
		ratio, err := strconv.ParseFloat(setting.OpenTelemetry.Traces.SamplerArg, 64)
		if err != nil {
			ratio = 1
		}
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	case ParentBasedAlwaysOff:
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case ParentBasedAlwaysOn:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

type otelErrorHandler struct{}

func (o otelErrorHandler) Handle(err error) {
	log.Error("internal opentelemetry error was raised: %s", err)
}
