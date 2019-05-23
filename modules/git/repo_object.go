// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"bytes"
	"io"
	"strings"
)

// ObjectType git object type
type ObjectType string

const (
	// ObjectCommit commit object type
	ObjectCommit ObjectType = "commit"
	// ObjectTree tree object type
	ObjectTree ObjectType = "tree"
	// ObjectBlob blob object type
	ObjectBlob ObjectType = "blob"
	// ObjectTag tag object type
	ObjectTag ObjectType = "tag"
)

// HashObject takes a reader and returns SHA1 hash for that reader
func (repo *Repository) HashObject(reader io.Reader) (SHA1, error) {
	idStr, err := repo.hashObject(reader)
	if err != nil {
		return SHA1{}, err
	}
	return NewIDFromString(idStr)
}

func (repo *Repository) hashObject(reader io.Reader) (string, error) {
	cmd := NewCommand("hash-object", "-w", "--stdin")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	err := cmd.RunInDirFullPipeline(repo.Path, stdout, stderr, reader)

	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}
