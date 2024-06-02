// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

// Opentelemetry settings
var OpenTelemetry = struct {
	Enabled      bool    // Toggle feature flag
	Address      string  // Upstream address of otlp collector
	Insecure     bool    // Allows to disable gRPC security options
	SamplerType  string  // Controls sampler, allowed options are "always" which sets sampler to always sample, "never" which sets to never sample, or ratio which samples based on a user provided ratio
	SamplerParam float64 // Ratio for "ratio" sampler type, allowed values are in range 0.0-1.0, values outside the range are treated as 0 or 1, whichever is closer
}{
	Enabled:      false,
	Address:      "localhost:4317",
	Insecure:     false,
	SamplerType:  "always",
	SamplerParam: 1,
}

func loadOpenTelemetryFrom(rootCfg ConfigProvider) {
	mustMapSetting(rootCfg, "opentelemetry", &OpenTelemetry)
}
