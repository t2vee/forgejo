// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package npm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

func TestParsePackage(t *testing.T) {
	packageName := "@scope/test-package"
	packageVersion := "1.0.1-pre"
	packageAuthor := "KN4CK3R"
	packageDescription := "Test Description"
	data := "H4sIAAAAAAAA/ytITM5OTE/VL4DQelnF+XkMVAYGBgZmJiYK2MRBwNDcSIHB2NTMwNDQzMwAqA7IMDUxA9LUdgg2UFpcklgEdAql5kD8ogCnhwio5lJQUMpLzE1VslJQcihOzi9I1S9JLS7RhSYIJR2QgrLUouLM/DyQGkM9Az1D3YIiqExKanFyUWZBCVQ2BKhVwQVJDKwosbQkI78IJO/tZ+LsbRykxFXLNdA+HwWjYBSMgpENACgAbtAACAAA"
	integrity := "sha512-yA4FJsVhetynGfOC1jFf79BuS+jrHbm0fhh+aHzCQkOaOBXKf9oBnC4a6DnLLnEsHQDRLYd00cwj8sCXpC+wIg=="

	t.Run("InvalidUpload", func(t *testing.T) {
		p, err := ParsePackage(bytes.NewReader([]byte{0}))
		assert.Nil(t, p)
		assert.Error(t, err)
	})

	t.Run("InvalidUploadNoData", func(t *testing.T) {
		b, _ := jsoniter.Marshal(packageUpload{})
		p, err := ParsePackage(bytes.NewReader(b))
		assert.Nil(t, p)
		assert.ErrorIs(t, err, ErrInvalidPackage)
	})

	t.Run("InvalidPackageName", func(t *testing.T) {
		name := " test "
		b, _ := jsoniter.Marshal(packageUpload{
			PackageMetadata: PackageMetadata{
				ID:   name,
				Name: name,
				Versions: map[string]*PackageMetadataVersion{
					packageVersion: {
						Name: name,
					},
				},
			},
		})

		p, err := ParsePackage(bytes.NewReader(b))
		assert.Nil(t, p)
		assert.ErrorIs(t, err, ErrInvalidPackageName)
	})

	t.Run("InvalidPackageVersion", func(t *testing.T) {
		version := "first-version"
		b, _ := jsoniter.Marshal(packageUpload{
			PackageMetadata: PackageMetadata{
				ID:   packageName,
				Name: packageName,
				Versions: map[string]*PackageMetadataVersion{
					version: {
						Name: packageName,
					},
				},
			},
		})

		p, err := ParsePackage(bytes.NewReader(b))
		assert.Nil(t, p)
		assert.ErrorIs(t, err, ErrInvalidPackageVersion)
	})

	t.Run("InvalidAttachment", func(t *testing.T) {
		b, _ := jsoniter.Marshal(packageUpload{
			PackageMetadata: PackageMetadata{
				ID:   packageName,
				Name: packageName,
				Versions: map[string]*PackageMetadataVersion{
					packageVersion: {
						Name: packageName,
					},
				},
			},
			Attachments: map[string]*PackageAttachment{
				"dummy.tgz": {},
			},
		})

		p, err := ParsePackage(bytes.NewReader(b))
		assert.Nil(t, p)
		assert.ErrorIs(t, err, ErrInvalidAttachment)
	})

	t.Run("InvalidData", func(t *testing.T) {
		filename := fmt.Sprintf("%s-%s.tgz", packageName, packageVersion)
		b, _ := jsoniter.Marshal(packageUpload{
			PackageMetadata: PackageMetadata{
				ID:   packageName,
				Name: packageName,
				Versions: map[string]*PackageMetadataVersion{
					packageVersion: {
						Name: packageName,
					},
				},
			},
			Attachments: map[string]*PackageAttachment{
				filename: {
					Data: "/",
				},
			},
		})

		p, err := ParsePackage(bytes.NewReader(b))
		assert.Nil(t, p)
		assert.ErrorIs(t, err, ErrInvalidAttachment)
	})

	t.Run("InvalidIntegrity", func(t *testing.T) {
		filename := fmt.Sprintf("%s-%s.tgz", packageName, packageVersion)
		b, _ := jsoniter.Marshal(packageUpload{
			PackageMetadata: PackageMetadata{
				ID:   packageName,
				Name: packageName,
				Versions: map[string]*PackageMetadataVersion{
					packageVersion: {
						Name: packageName,
						Dist: PackageDistribution{
							Integrity: "sha512-test==",
						},
					},
				},
			},
			Attachments: map[string]*PackageAttachment{
				filename: {
					Data: data,
				},
			},
		})

		p, err := ParsePackage(bytes.NewReader(b))
		assert.Nil(t, p)
		assert.ErrorIs(t, err, ErrInvalidIntegrity)
	})

	t.Run("InvalidIntegrity2", func(t *testing.T) {
		filename := fmt.Sprintf("%s-%s.tgz", packageName, packageVersion)
		b, _ := jsoniter.Marshal(packageUpload{
			PackageMetadata: PackageMetadata{
				ID:   packageName,
				Name: packageName,
				Versions: map[string]*PackageMetadataVersion{
					packageVersion: {
						Name: packageName,
						Dist: PackageDistribution{
							Integrity: integrity,
						},
					},
				},
			},
			Attachments: map[string]*PackageAttachment{
				filename: {
					Data: base64.StdEncoding.EncodeToString([]byte("data")),
				},
			},
		})

		p, err := ParsePackage(bytes.NewReader(b))
		assert.Nil(t, p)
		assert.ErrorIs(t, err, ErrInvalidIntegrity)
	})

	t.Run("Valid", func(t *testing.T) {
		filename := fmt.Sprintf("%s-%s.tgz", packageName, packageVersion)
		b, _ := jsoniter.Marshal(packageUpload{
			PackageMetadata: PackageMetadata{
				ID:   packageName,
				Name: packageName,
				Versions: map[string]*PackageMetadataVersion{
					packageVersion: {
						Name:        packageName,
						Version:     packageVersion,
						Description: packageDescription,
						Author:      User{Name: packageAuthor},
						License:     "MIT",
						Homepage:    "https://gitea.io/",
						Readme:      packageDescription,
						Dependencies: map[string]string{
							"package": "1.2.0",
						},
						Dist: PackageDistribution{
							Integrity: integrity,
						},
					},
				},
			},
			Attachments: map[string]*PackageAttachment{
				filename: {
					Data: data,
				},
			},
		})

		p, err := ParsePackage(bytes.NewReader(b))
		assert.NotNil(t, p)
		assert.NoError(t, err)

		assert.Equal(t, packageName, p.Name)
		assert.Equal(t, packageVersion, p.Version)
		assert.Equal(t, fmt.Sprintf("%s-%s.tgz", strings.Split(packageName, "/")[1], packageVersion), p.Filename)
		b, _ = base64.StdEncoding.DecodeString(data)
		assert.Equal(t, b, p.Data)
		assert.Equal(t, packageDescription, p.Metadata.Description)
		assert.Equal(t, packageDescription, p.Metadata.Readme)
		assert.Equal(t, packageAuthor, p.Metadata.Author)
		assert.Equal(t, "MIT", p.Metadata.License)
		assert.Equal(t, "https://gitea.io/", p.Metadata.ProjectURL)
		assert.Contains(t, p.Metadata.Dependencies, "package")
		assert.Equal(t, "1.2.0", p.Metadata.Dependencies["package"])
	})
}
