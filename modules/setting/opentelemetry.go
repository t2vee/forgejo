// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"code.gitea.io/gitea/modules/log"
)

const (
	opentelemetrySectionName            = "opentelemetry"
	opentelemetryTraceSubSectionName    = "traces"
	opentelemetryResourceSubSectionName = "resources"
)

// Opentelemetry settings
var (
	OpenTelemetry = struct {
		Enabled  bool // turned on if any part of the feature is active - activation being marked by non-nil endpoint
		Traces   traceConfig
		Resource resourceConfig
	}{
		Traces:   traceConfig{Timeout: 10 * time.Second, Sampler: "parentbased_always_on", SamplerArg: "1.0"},
		Resource: resourceConfig{ServiceName: "forgejo", EnabledDecoders: "all"},
	}
)

type traceConfig struct {
	Endpoint          *url.URL // A base endpoint URL for any signal type, with an optionally-specified port number
	Headers           map[string]string
	Insecure          bool          // Disable TLS
	Compression       string        // Supported value - ""/"gzip"
	Timeout           time.Duration // The timeout value for all outgoing data
	Sampler           string
	SamplerArg        string
	Certificate       string
	ClientKey         string
	ClientCertificate string
}
type resourceConfig struct {
	ServiceName     string // Value of the service.name resource attribute, defaults to APP_NAME in lowercase
	Attributes      string // unprocessed attributes for the resource
	EnabledDecoders string
}

func loadOpenTelemetryFrom(rootCfg ConfigProvider) {

	sec := rootCfg.Section(opentelemetrySectionName)
	traceSec := rootCfg.Section(opentelemetrySectionName + "." + opentelemetryTraceSubSectionName)
	resourceSec := rootCfg.Section(opentelemetrySectionName + "." + opentelemetryResourceSubSectionName)
	loadResourceConfig(resourceSec)
	loadTraceConfig(sec, traceSec)
	OpenTelemetry.Enabled = OpenTelemetry.Traces.Endpoint != nil
}

func loadResourceConfig(sec ConfigSection) {
	OpenTelemetry.Resource.ServiceName = sec.Key("SERVICE_NAME").MustString(OpenTelemetry.Resource.ServiceName)
	OpenTelemetry.Resource.Attributes = sec.Key("RESOURCE_ATTRIBUTES").String()
	OpenTelemetry.Resource.EnabledDecoders = sec.Key("ENABLE_DECODERS").MustString(OpenTelemetry.Resource.EnabledDecoders)
	OpenTelemetry.Resource.EnabledDecoders = strings.ToLower(strings.TrimSpace(OpenTelemetry.Resource.EnabledDecoders))
}

func loadTraceConfig(rootSec, traceSec ConfigSection) {
	if !rootSec.HasKey("ENDPOINT") && !traceSec.HasKey("ENDPOINT") {
		return
	}
	endpoint := traceSec.Key("ENDPOINT").MustString(rootSec.Key("ENDPOINT").String())
	if ep, err := url.Parse(endpoint); err != nil && ep.Host != "" {
		OpenTelemetry.Traces.Endpoint = ep
	} else {
		log.Warn("Otel trace endpoint parsing failure, disabaling traces.")
		return
	}
	OpenTelemetry.Traces.Insecure = traceSec.Key("INSECURE").MustBool(rootSec.Key("INSECURE").MustBool(OpenTelemetry.Traces.Insecure))
	OpenTelemetry.Traces.Compression = traceSec.Key("COMPRESSION").In(rootSec.Key("COMPRESSION").In(OpenTelemetry.Traces.Compression, []string{"gzip"}), []string{"gzip"})
	OpenTelemetry.Traces.Timeout = traceSec.Key("TIMEOUT").MustDuration(rootSec.Key("TIMEOUT").MustDuration(OpenTelemetry.Traces.Timeout))
	OpenTelemetry.Traces.Sampler = traceSec.Key("SAMPLER").MustString(OpenTelemetry.Traces.Sampler)
	OpenTelemetry.Traces.SamplerArg = traceSec.Key("SAMPLER_ARG").MustString(OpenTelemetry.Traces.Sampler)
	headers := rootSec.Key("HEADERS").String()
	if headers != "" {
		for k, v := range _stringToHeader(headers) {
			OpenTelemetry.Traces.Headers[k] = v
		}
	}
	headers = traceSec.Key("HEADERS").String()
	if headers != "" {
		for k, v := range _stringToHeader(headers) {
			OpenTelemetry.Traces.Headers[k] = v
		}
	}

	OpenTelemetry.Traces.Certificate = rootSec.Key("CERTIFICATE").MustString(rootSec.Key("CERTIFICATE").String())
	OpenTelemetry.Traces.ClientCertificate = rootSec.Key("CLIENT_CERTIFICATE").MustString(rootSec.Key("CLIENT_CERTIFICATE").String())
	OpenTelemetry.Traces.ClientKey = rootSec.Key("CLIENT_KEY").MustString(rootSec.Key("CLIENT_KEY").String())
	if len(OpenTelemetry.Traces.Certificate) > 0 && !filepath.IsAbs(OpenTelemetry.Traces.Certificate) {
		OpenTelemetry.Traces.Certificate = filepath.Join(CustomPath, OpenTelemetry.Traces.Certificate)
	}
	if len(OpenTelemetry.Traces.ClientCertificate) > 0 && !filepath.IsAbs(OpenTelemetry.Traces.ClientCertificate) {
		OpenTelemetry.Traces.ClientCertificate = filepath.Join(CustomPath, OpenTelemetry.Traces.ClientCertificate)
	}
	if len(OpenTelemetry.Traces.ClientKey) > 0 && !filepath.IsAbs(OpenTelemetry.Traces.ClientKey) {
		OpenTelemetry.Traces.ClientKey = filepath.Join(CustomPath, OpenTelemetry.Traces.ClientKey)
	}
}

// Opentelemetry SDK function port

func _stringToHeader(value string) map[string]string {
	headersPairs := strings.Split(value, ",")
	headers := make(map[string]string)

	for _, header := range headersPairs {
		n, v, found := strings.Cut(header, "=")
		if !found {
			log.Warn("Otel header ignored, err=\"missing '='\", input=%s", header)
			continue
		}
		name, err := url.PathUnescape(n)
		if err != nil {
			log.Warn("Otel header ignored, err=\"escape header key\", key=%s", n)
			continue
		}
		trimmedName := strings.TrimSpace(name)
		value, err := url.PathUnescape(v)
		if err != nil {
			log.Warn("Otel header ignored, err=\"escape header value\", value=%s", v)
			continue
		}
		trimmedValue := strings.TrimSpace(value)

		headers[trimmedName] = trimmedValue
	}

	return headers
}
