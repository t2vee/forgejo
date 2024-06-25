// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: EUPL-1.2

package setting

// Quota settings
var Quota = struct {
	Enabled       bool     `ini:"ENABLED"`
	DefaultGroups []string `ini:"DEFAULT_GROUPS"`
}{
	Enabled:       false,
	DefaultGroups: []string{},
}

func loadQuotaFrom(rootCfg ConfigProvider) {
	mustMapSetting(rootCfg, "quota", &Quota)
}
