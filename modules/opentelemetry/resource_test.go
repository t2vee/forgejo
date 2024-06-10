// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package opentelemetry

import (
	"context"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
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
	defer test.MockVariableValue(&setting.OpenTelemetry.Resource.Attributes, "Iuse=Arch,bythe=way")()
	res, err := newResource(ctx)
	assert.NoError(t, err)
	expected, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(setting.OpenTelemetry.Resource.ServiceName),
		semconv.ServiceVersion(setting.ForgejoVersion),
		attribute.String("Iuse", "Arch"),
		attribute.String("bythe", "way"),
	))
	assert.NoError(t, err)
	assert.Equal(t, res, expected)
}
