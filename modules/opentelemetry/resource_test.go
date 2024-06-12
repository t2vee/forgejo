// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

func TestResourceServiceName(t *testing.T) {
	ctx := context.Background()

	resource, err := newResource(ctx)
	assert.NoError(t, err)
	serviceKeyPresant := false
	for _, v := range resource.Attributes() {
		if v.Key == semconv.ServiceNameKey {
			assert.Equal(t, "forgejo", v.Value.AsString())
			serviceKeyPresant = true
		}
	}
	assert.True(t, serviceKeyPresant)
	serviceKeyPresant = false
	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.ServiceName, "non-default value")()
	resource, err = newResource(ctx)
	assert.NoError(t, err)
	for _, v := range resource.Attributes() {
		if v.Key == semconv.ServiceNameKey {
			assert.Equal(t, "non-default value", v.Value.AsString())
			serviceKeyPresant = true
		}
	}
	assert.True(t, serviceKeyPresant)
}

func TestResourceAttributes(t *testing.T) {
	ctx := context.Background()
	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.EnabledDecoders, "")()
	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.Attributes, "Test=LABEL,broken,unescape=%XXlabel")()
	res, err := newResource(ctx)
	assert.Error(t, err)
	expected, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(setting.OpenTelemetry.Resource.ServiceName),
		semconv.ServiceVersion(setting.ForgejoVersion),
	))
	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestDecoderParity(t *testing.T) {
	ctx := context.Background()
	defer test.MockProtect(&setting.OpenTelemetry.Resource.EnabledDecoders)()
	setting.OpenTelemetry.Resource.EnabledDecoders = "all"
	res, err := newResource(ctx)
	assert.NoError(t, err)
	setting.OpenTelemetry.Resource.EnabledDecoders = "sdk,process,os,host"
	res2, err := newResource(ctx)
	assert.NoError(t, err)
	assert.Equal(t, res, res2)
}
