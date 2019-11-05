// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package util

import (
	"os"
	"path/filepath"
)

// EnsureAbsolutePath ensure that a path is absolute, making it
// relative to absoluteBase if necessary
func EnsureAbsolutePath(path string, absoluteBase string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(absoluteBase, path)
}

// GetDirectorySize returns the dumb disk consumption for a given path
func GetDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info != nil && !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
