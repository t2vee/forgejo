// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"google.golang.org/grpc"
)

func TestNoopDefault(t *testing.T) {
	inMem := tracetest.NewInMemoryExporter()
	called := false
	exp := func(ctx context.Context) (sdktrace.SpanExporter, error) {
		called = true
		return inMem, nil
	}
	defer test.MockVariableValue(&newTraceExporter, exp)()
	ctx := context.Background()
	defer Setup(ctx)(ctx)
	tracer := otel.Tracer("test_noop")

	_, span := tracer.Start(ctx, "test span")

	assert.False(t, span.SpanContext().HasTraceID())
	assert.False(t, span.SpanContext().HasSpanID())
	assert.False(t, called)
}

func TestOtelIntegration(t *testing.T) {
	grpcMethods := make(chan string)
	collector := grpc.NewServer(grpc.UnknownServiceHandler(func(srv any, stream grpc.ServerStream) error {
		method, _ := grpc.Method(stream.Context())
		grpcMethods <- method
		return nil
	}))
	t.Cleanup(collector.GracefulStop)
	ln, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	t.Cleanup(func() {
		ln.Close()
	})
	go collector.Serve(ln)

	traceEndpoint, err := url.Parse("http://" + ln.Addr().String())
	assert.NoError(t, err)

	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.ServiceName, "forgejo-integration")()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Endpoint, traceEndpoint)()
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Insecure, true)()
	ctx := context.Background()
	defer Setup(ctx)(ctx)

	tracer := otel.Tracer("test_jaeger")
	_, span := tracer.Start(ctx, "test span")
	assert.True(t, span.SpanContext().HasTraceID())
	assert.True(t, span.SpanContext().HasSpanID())

	span.End()
	// Give the exporter time to send the span
	select {
	case method := <-grpcMethods:
		assert.Equal(t, "/opentelemetry.proto.collector.trace.v1.TraceService/Export", method)
	case <-time.After(10 * time.Second):
		t.Fatal("no grpc call within 10s")
	}
}

func TestExporter(t *testing.T) {
	inMem := tracetest.NewInMemoryExporter()
	exp := func(ctx context.Context) (sdktrace.SpanExporter, error) {
		return inMem, nil
	}
	defer test.MockVariableValue(&newTraceExporter, exp)()

	// Force feature activation
	endpoint, err := url.Parse("http://localhost:4317")
	assert.NoError(t, err)
	defer test.MockVariableValue(&setting.OpenTelemetry.Traces.Endpoint, endpoint)()

	ctx := context.Background()
	defer Setup(ctx)(ctx)
	assert.NoError(t, err)
	tracer := otel.Tracer("test_grpc")

	_, span := tracer.Start(ctx, "test span")

	assert.True(t, span.SpanContext().HasTraceID())
	assert.True(t, span.SpanContext().HasSpanID())
}
