// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package lfs

import (
	"errors"
	"io"
	"net/http"
)

type TransferAdapter interface {
	Name() string
	Download(r *Representation) (io.ReadCloser, error)
	//Upload(reader io.Reader) error
 }

type BasicTransferAdapter struct {
	client *http.Client
}

func (a *BasicTransferAdapter) Name() string {
	return "basic"
}

func (a *BasicTransferAdapter) Download(r *Representation) (io.ReadCloser, error) {
	download, ok := r.Actions["download"]
	if !ok {
		return nil, errors.New("Action 'download' not found")
	}

	req, err := http.NewRequest("GET", download.Href, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range download.Header {
		req.Header.Set(key, value)
	}
	
	res, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}
