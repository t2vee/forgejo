// Copyright 2020 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package markdown

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v2"
)

func isYAMLSeparator(line string) bool {
	line = strings.TrimSpace(line)
	for i := 0; i < len(line); i++ {
		if line[i] != '-' {
			return false
		}
	}
	return len(line) > 2
}

// ExtractMetadata consumes a markdown file, parses YAML frontmatter,
// and returns the frontmatter metadata separated from the markdown content
func ExtractMetadata(contents string, out interface{}) (string, error) {
	var front, body []string
	seps := make([]int, 0, 2)
	lines := strings.Split(contents, "\n")
	for idx, line := range lines {
		if len(seps) == 2 {
			// Leave the separator lines out of the parsing
			front, body = lines[seps[0]+1:seps[1]], lines[seps[1]+1:]
			break
		}
		if isYAMLSeparator(line) {
			seps = append(seps, idx)
			continue
		}
	}

	if len(front) == 0 && len(body) == 0 {
		return "", errors.New("could not determine metadata")
	}

	if err := yaml.Unmarshal([]byte(strings.Join(front, "\n")), out); err != nil {
		return "", err
	}
	return strings.Join(body, "\n"), nil
}
