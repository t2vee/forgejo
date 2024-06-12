// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

const (
	decoderTelemetrySdk = "sdk"
	decoderProcess      = "process"
	decoderOS           = "os"
	decoderHost         = "host"
)

func newResource(ctx context.Context) (*resource.Resource, error) {
	opts := []resource.Option{resource.WithDetectors(fromSettings{})}

	opts = append(opts, parseDecoderOpts()...)
	opts = append(opts, resource.WithAttributes(
		semconv.ServiceName(setting.OpenTelemetry.Resource.ServiceName), semconv.ServiceVersion(setting.ForgejoVersion),
	))
	return resource.New(ctx, opts...)
}

func parseDecoderOpts() []resource.Option {
	if setting.OpenTelemetry.Resource.EnabledDecoders == "all" {
		return []resource.Option{
			resource.WithTelemetrySDK(),
			resource.WithProcess(),
			resource.WithOS(),
			resource.WithHost(),
		}
	}
	var opts []resource.Option
	for _, v := range strings.Split(setting.OpenTelemetry.Resource.EnabledDecoders, ",") {
		switch v {
		case decoderTelemetrySdk:
			opts = append(opts, resource.WithTelemetrySDK())
		case decoderProcess:
			opts = append(opts, resource.WithProcess())
		case decoderOS:
			opts = append(opts, resource.WithOS())
		case decoderHost:
			opts = append(opts, resource.WithHost())
		}
	}
	return opts
}

type fromSettings struct{}

func (fromSettings) Detect(context.Context) (*resource.Resource, error) {
	rawAttrs := strings.TrimSpace(setting.OpenTelemetry.Resource.Attributes)

	if rawAttrs == "" {
		return resource.Empty(), nil
	}

	pairs := strings.Split(rawAttrs, ",")
	var attrs []attribute.KeyValue

	var invalid []string
	for _, p := range pairs {
		k, v, found := strings.Cut(p, "=")
		if !found {
			invalid = append(invalid, p)
			continue
		}
		key := strings.TrimSpace(k)
		val, err := url.PathUnescape(strings.TrimSpace(v))
		if err != nil {
			// Retain original value if decoding fails, otherwise it will be
			// an empty string.
			val = v
			log.Warn("Otel resource attribute decoding error, retaining unescaped value. key=%s, val=%s", key, val)
		}
		attrs = append(attrs, attribute.String(key, val))
	}
	var err error
	if len(invalid) > 0 {
		err = fmt.Errorf("%s: %v", "Partial resource: missing values", invalid)
	}

	res := resource.NewSchemaless(attrs...)

	return res, err
}
