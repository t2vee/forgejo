// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/url"
	"os"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
	// "go.opentelemetry.io/otel/sdk/resource"
)

const (
	opentelemetrySectionName = "opentelemetry"
	traceSubSectionName      = "traces"
	resourceSubSectionName   = "resources"
)

// Opentelemetry settings
var (
	OpenTelemetry = struct {
		Traces   traceConfig
		Resource resourceConfig
	}{
		Traces:   traceConfig{Timeout: 10 * time.Second, Sampler: "parentbased_always_on"},
		Resource: resourceConfig{ServiceName: "forgejo"},
	}
)

type traceConfig struct {
	Endpoint          string            // A base endpoint URL for any signal type, with an optionally-specified port number
	Certificate       string            // Not implemented
	ClientCertificate string            // Not implemented
	ClientKey         string            // Not implemented
	Insecure          bool              // Disable TLS
	Headers           map[string]string // A list of headers to apply to outgoing data.
	Compression       string            // Supported value - ""/"gzip"
	Timeout           time.Duration     // The timeout value for all outgoing data
	Sampler           string
	SamplerArg        float64
}
type resourceConfig struct {
	ServiceName string // Value of the service.name resource attribute, defaults to APP_NAME in lowercase
	// ResourceAttributes []attribute.KeyValue // Not implemented, Key-value pairs to be used as resource attributes
	EnabledDetectors []string // Not implemented
}

func loadOpenTelemetryFrom(rootCfg ConfigProvider) {
	// Load generic config
	sec, _ := rootCfg.GetSection(opentelemetrySectionName)
	if sec != nil {
		loadTraceConfig(sec)
	}
	// Load resource
	resourceSec, _ := rootCfg.GetSection(opentelemetrySectionName + "." + resourceSubSectionName)
	if resourceSec != nil {
		loadResourceConfig(resourceSec)
	}

	// Override with more domain specific config
	sec, _ = rootCfg.GetSection(opentelemetrySectionName + "." + traceSubSectionName)
	if sec != nil {
		loadTraceConfig(sec)
	}
}

func loadResourceConfig(sec ConfigSection) {
	OpenTelemetry.Resource.ServiceName = sec.Key("SERVICE_NAME").MustString(OpenTelemetry.Resource.ServiceName)
	os.Setenv("OTEL_RESOURCE_ATTRIBUTES", sec.Key("RESOURCE_ATTRIBUTES").MustString("")) // Let otel handle resource attributes

}

func loadTraceConfig(sec ConfigSection) {
	OpenTelemetry.Traces.Endpoint = sec.Key("ENDPOINT").MustString(OpenTelemetry.Traces.Endpoint)
	OpenTelemetry.Traces.Insecure = sec.Key("INSECURE").MustBool(OpenTelemetry.Traces.Insecure)
	for k, v := range stringToHeader(sec.Key("HEADERS").MustString("")) {
		OpenTelemetry.Traces.Headers[k] = v
	}
	OpenTelemetry.Traces.Compression = sec.Key("COMPRESSION").MustString(OpenTelemetry.Traces.Compression)
	OpenTelemetry.Traces.Timeout = sec.Key("TIMEOUT").MustDuration(OpenTelemetry.Traces.Timeout)
}

// Port of internal otlp function
func stringToHeader(value string) map[string]string {
	headerPairs := strings.Split(value, ",")
	headers := make(map[string]string)
	if value == "" {
		return headers
	}

	for _, header := range headerPairs {
		n, v, found := strings.Cut(header, "=")
		if !found {
			log.Warn("Parsing opentelemetry header failed, ignoring header: %s", header)
			continue
		}
		name, err := url.PathUnescape(n)
		if err != nil {
			log.Warn("Parsing opentelemetry header key failed, ignoring header: %s", header)
			continue
		}
		trimmedName := strings.TrimSpace(name)
		value, err := url.PathUnescape(v)
		if err != nil {
			log.Warn("Parsing opentelemetry header value failed, ignoring header: %s", header)
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		headers[trimmedName] = trimmedValue

	}
	return headers
}
