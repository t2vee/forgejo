package opentelemetry_test

import (
	"context"
	"testing"

	"code.gitea.io/gitea/modules/opentelemetry"
	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
)

func TestTraceDisabled(t *testing.T) {
	ctx := context.Background()
	shutdown, err := opentelemetry.SetupOTel(ctx)
	defer func() {
		shutdown(ctx)
	}()
	if assert.NoError(t, err) {
		_, span := opentelemetry.Tracer.Start(ctx, "debug")
		assert.False(t, span.IsRecording())
	}
}

func TestTraceEnabled(t *testing.T) {
	ctx := context.Background()
	setting.OpenTelemetry.Enabled = true
	shutdown, err := opentelemetry.SetupOTel(ctx)
	defer func() {
		shutdown(ctx)
	}()
	if assert.NoError(t, err) {
		_, span := opentelemetry.Tracer.Start(ctx, "debug")
		assert.True(t, span.IsRecording())
	}
}
