// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

// check that files stored in LFS render properly in the web UI
func TestLFSRender(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	// check that a markup file is flagged with "Stored in Git LFS" and shows its text
	t.Run("Markup", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/CONTRIBUTING.md")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		fileInfo := doc.Find("div.file-info-entry").First().Text()
		assert.Contains(t, fileInfo, "Stored with Git LFS")

		content := doc.Find("div.file-view").Text()
		assert.Contains(t, content, "Testing documents in LFS")
	})

	// check that an image is flagged with "Stored in Git LFS" and renders inline
	t.Run("Image", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/jpeg.jpg")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		fileInfo := doc.Find("div.file-info-entry").First().Text()
		assert.Contains(t, fileInfo, "Stored with Git LFS")

		src, exists := doc.Find(".file-view img").Attr("src")
		assert.True(t, exists, "The image should be in an <img> tag")
		assert.Equal(t, "/user2/lfs/media/branch/master/jpeg.jpg", src, "The image should use the /media link because it's in LFS")
	})

	// check that a binary file is flagged with "Stored in Git LFS" and renders a /media/ link instead of a /raw/ link
	t.Run("Binary", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/crypt.bin")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		fileInfo := doc.Find("div.file-info-entry").First().Text()
		assert.Contains(t, fileInfo, "Stored with Git LFS")

		rawLink, exists := doc.Find("div.file-view > div.view-raw > a").Attr("href")
		assert.True(t, exists, "Download link should render instead of content because this is a binary file")
		assert.Equal(t, "/user2/lfs/media/branch/master/crypt.bin", rawLink, "The download link should use the proper /media link because it's in LFS")
	})

	// check that a directory with a README file shows its text
	t.Run("Readme", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/lfs/src/branch/master/subdir")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body).doc

		content := doc.Find("div.file-view").Text()
		assert.Contains(t, content, "Testing READMEs in LFS")
	})

	t.Run("/settings/lfs/pointers", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// visit /user2/lfs/settings/lfs/pointer
		req := NewRequest(t, "GET", "/user2/lfs/settings/lfs/pointers")
		resp := session.MakeRequest(t, req, http.StatusOK)

		// follow the first link to /user2/lfs/settings/lfs/find?oid=....
		filesTable := NewHTMLParser(t, resp.Body).doc.Find("#lfs-files-table")
		assert.Contains(t, filesTable.Text(), "Find commits")
		lfsFind := filesTable.Find(`.primary.button[href^="/user2"]`)
		assert.Greater(t, lfsFind.Length(), 0)
		lfsFindPath, exists := lfsFind.First().Attr("href")
		assert.True(t, exists)

		assert.Contains(t, lfsFindPath, "oid=")
		req = NewRequest(t, "GET", lfsFindPath)
		resp = session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body).doc
		assert.Equal(t, 1, doc.Find(`.sha.label[href="/user2/lfs/commit/73cf03db6ece34e12bf91e8853dc58f678f2f82d"]`).Length(), "could not find link to commit")
	})
}
