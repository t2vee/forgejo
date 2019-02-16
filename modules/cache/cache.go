// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cache

import (
	"fmt"
	"strconv"

	"code.gitea.io/gitea/modules/setting"

	mc "github.com/go-macaron/cache"
)

// Cache defines the cache provider
var Cache mc.Cache

// NewContext start cache service
func NewContext() error {
	if setting.CacheService == nil || Cache != nil {
		return nil
	}

	var err error
	Cache, err = mc.NewCacher(setting.CacheService.Adapter, mc.Options{
		Adapter:       setting.CacheService.Adapter,
		AdapterConfig: setting.CacheService.Conn,
		Interval:      setting.CacheService.Interval,
	})
	return err
}

// GetInt returns key value from cache with callback when no key exists in cache
func GetInt(key string, getFunc func() (int, error)) (int, error) {
	if Cache == nil || setting.CacheService.TTL == 0 {
		return getFunc()
	}
	if !Cache.IsExist(key) {
		var (
			value int
			err   error
		)
		if value, err = getFunc(); err != nil {
			return value, err
		}
		Cache.Put(key, value, int64(setting.CacheService.TTL.Seconds()))
	}
	switch value := Cache.Get(key).(type) {
	case int:
		return value, nil
	case string:
		v, err := strconv.Atoi(value)
		if err != nil {
			return 0, err
		}
		return v, nil
	default:
		return 0, fmt.Errorf("Unsupported cached value type: %v", value)
	}
}

// GetInt64 returns key value from cache with callback when no key exists in cache
func GetInt64(key string, getFunc func() (int64, error)) (int64, error) {
	if Cache == nil || setting.CacheService.TTL == 0 {
		return getFunc()
	}
	if !Cache.IsExist(key) {
		var (
			value int64
			err   error
		)
		if value, err = getFunc(); err != nil {
			return value, err
		}
		Cache.Put(key, value, int64(setting.CacheService.TTL.Seconds()))
	}
	switch value := Cache.Get(key).(type) {
	case int64:
		return value, nil
	case string:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, err
		}
		return v, nil
	default:
		return 0, fmt.Errorf("Unsupported cached value type: %v", value)
	}
}

// Remove key from cache
func Remove(key string) {
	if Cache == nil {
		return
	}
	Cache.Delete(key)
}
