// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"

	"code.gitea.io/git"
	"code.gitea.io/gitea/modules/process"
)

// BlameFile represents while git blame output
type BlameFile struct {
	Parts []BlamePart
}

// BlamePart represents block of blame - continuous lines with one sha
type BlamePart struct {
	Sha   string
	Lines []string
}

// BlameReader returns part of file blame one by one
type BlameReader struct {
	cmd     *exec.Cmd
	pid     int64
	output  io.ReadCloser
	scanner *bufio.Scanner
	lastSha *string
}

// NextPart returns next part of blame (sequencial code lines with the same commit)
func (r *BlameReader) NextPart() (*BlamePart, error) {

	var blamePart *BlamePart

	scanner := r.scanner

	if r.lastSha != nil {
		blamePart = &BlamePart{*r.lastSha, make([]string, 0, 0)}
	}

	for scanner.Scan() {
		line := scanner.Text()

		lines := shaLineRegex.FindStringSubmatch(line)

		if len(line) == 0 {
		} else if lines != nil {

			sha1 := lines[1]

			if blamePart == nil {
				blamePart = &BlamePart{sha1, make([]string, 0, 0)}
			}

			if blamePart.Sha != sha1 {
				r.lastSha = &sha1
				return blamePart, nil
			}

		} else if line[0] == '\t' {

			code := line[1:]

			blamePart.Lines = append(blamePart.Lines, code)

		}

	}

	r.lastSha = nil

	return blamePart, nil
}

// Close BlameReader - don't run NextPart after invoking that
func (r *BlameReader) Close() error {
	process.GetManager().Remove(r.pid)

	if err := r.cmd.Wait(); err != nil {
		return fmt.Errorf("Wait: %v", err)
	}

	return nil

}

// CreateBlameReader creates reader for given repository, commit and file
func CreateBlameReader(repoPath, commitID, file string) (*BlameReader, error) {

	_, err := git.OpenRepository(repoPath)
	if err != nil {
		return nil, err
	}

	return createBlameReader(repoPath, "git", "blame", commitID, "--porcelain", "--", file)

}

func createBlameReader(dir string, command ...string) (*BlameReader, error) {

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("StdoutPipe: %v", err)
	}

	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("Start: %v", err)
	}

	pid := process.GetManager().Add(fmt.Sprintf("GetBlame [repo_path: %s]", dir), cmd)

	scanner := bufio.NewScanner(stdout)

	return &BlameReader{
		cmd,
		pid,
		stdout,
		scanner,
		nil,
	}, nil

}

var shaLineRegex = regexp.MustCompile("^([a-z0-9]{40})")

func parseBlameOutput(reader io.Reader) (*BlameFile, error) {

	var parts = make([]BlamePart, 0, 0)

	var blamePart *BlamePart

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		lines := shaLineRegex.FindStringSubmatch(line)

		if len(line) == 0 {
		} else if lines != nil {

			sha1 := lines[1]

			if blamePart == nil {
				blamePart = &BlamePart{sha1, make([]string, 0, 0)}
			}

			if blamePart.Sha != sha1 {
				parts = append(parts, *blamePart)
				blamePart = &BlamePart{sha1, make([]string, 0, 0)}
			}

		} else if line[0] == '\t' {

			code := line[1:]

			blamePart.Lines = append(blamePart.Lines, code)

		}

	}

	if blamePart != nil {
		parts = append(parts, *blamePart)
	}

	return &BlameFile{parts}, nil
}
