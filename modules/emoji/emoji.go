// Copyright 2020 The Gitea Authors. All rights reserved.
// Copyright 2015 Kenneth Shaw
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package emoji

import "code.gitea.io/gitea/traceinit"

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Gemoji is a set of emoji data.
type Gemoji []Emoji

// Emoji represents a single emoji and associated data.
type Emoji struct {
	Emoji          string
	Description    string
	Aliases        []string
	UnicodeVersion string
}

var (
	// codeMap provides a map of the emoji unicode code to its emoji data.
	codeMap map[string]int

	// aliasMap provides a map of the alias to its emoji data.
	aliasMap map[string]int

	// codeReplacer is the string replacer for emoji codes.
	codeReplacer *strings.Replacer

	// aliasReplacer is the string replacer for emoji aliases.
	aliasReplacer *strings.Replacer

	once sync.Once
)

func loadMap() {

	once.Do(func() {
		traceinit.Trace("./modules/emoji/emoji.go")
		// initialize
		start := time.Now()
		fmt.Printf("%v emoji init()\n", time.Now().Format("15:04:05.000000"))
		codeMap = make(map[string]int, len(GemojiData))
		aliasMap = make(map[string]int, len(GemojiData))

		// process emoji codes and aliases
		codePairs := make([]string, 0)
		aliasPairs := make([]string, 0)
		for i, e := range GemojiData {
			if e.Emoji == "" || len(e.Aliases) == 0 {
				continue
			}

			// setup codes
			codeMap[e.Emoji] = i
			codePairs = append(codePairs, e.Emoji, ":"+e.Aliases[0]+":")

			// setup aliases
			for _, a := range e.Aliases {
				if a == "" {
					continue
				}

				aliasMap[a] = i
				aliasPairs = append(aliasPairs, ":"+a+":", e.Emoji)
			}
		}

		// create replacers
		codeReplacer = strings.NewReplacer(codePairs...)
		aliasReplacer = strings.NewReplacer(aliasPairs...)
		fmt.Printf("\ttime taken: %v\n\n", time.Since(start))

	})
}

// FromCode retrieves the emoji data based on the provided unicode code (ie,
// "\u2618" will return the Gemoji data for "shamrock").
func FromCode(code string) *Emoji {
	loadMap()
	i, ok := codeMap[code]
	if !ok {
		return nil
	}

	return &GemojiData[i]
}

// FromAlias retrieves the emoji data based on the provided alias in the form
// "alias" or ":alias:" (ie, "shamrock" or ":shamrock:" will return the Gemoji
// data for "shamrock").
func FromAlias(alias string) *Emoji {
	loadMap()
	if strings.HasPrefix(alias, ":") && strings.HasSuffix(alias, ":") {
		alias = alias[1 : len(alias)-1]
	}

	i, ok := aliasMap[alias]
	if !ok {
		return nil
	}

	return &GemojiData[i]
}

// ReplaceCodes replaces all emoji codes with the first corresponding emoji
// alias (in the form of ":alias:") (ie, "\u2618" will be converted to
// ":shamrock:").
func ReplaceCodes(s string) string {
	loadMap()
	return codeReplacer.Replace(s)
}

// ReplaceAliases replaces all aliases of the form ":alias:" with its
// corresponding unicode value.
func ReplaceAliases(s string) string {
	loadMap()
	return aliasReplacer.Replace(s)
}
