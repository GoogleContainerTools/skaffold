// Copyright 2024 The Tessera authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/transparency-dev/tessera/api/layout"
	"github.com/transparency-dev/tessera/internal/fetcher"
	"k8s.io/klog/v2"
)

// NewHTTPFetcher creates a new HTTPFetcher for the log rooted at the given URL, using
// the provided HTTP client.
//
// rootURL should end in a trailing slash.
// c may be nil, in which case http.DefaultClient will be used.
func NewHTTPFetcher(rootURL *url.URL, c *http.Client) (*HTTPFetcher, error) {
	if !strings.HasSuffix(rootURL.String(), "/") {
		rootURL.Path += "/"
	}
	if c == nil {
		c = http.DefaultClient
	}
	return &HTTPFetcher{
		c:       c,
		rootURL: rootURL,
	}, nil
}

// HTTPFetcher knows how to fetch log artifacts from a log being served via HTTP.
type HTTPFetcher struct {
	c          *http.Client
	rootURL    *url.URL
	authHeader string
}

// SetAuthorizationHeader sets the value to be used with an Authorization: header
// for every request made by this fetcher.
func (h *HTTPFetcher) SetAuthorizationHeader(v string) {
	h.authHeader = v
}

func (h HTTPFetcher) fetch(ctx context.Context, p string) ([]byte, error) {
	u, err := h.rootURL.Parse(p)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext(%q): %v", u.String(), err)
	}
	if h.authHeader != "" {
		req.Header.Add("Authorization", h.authHeader)
	}
	r, err := h.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get(%q): %v", u.String(), err)
	}
	switch r.StatusCode {
	case http.StatusOK:
		// All good, continue below
	case http.StatusNotFound:
		// Need to return ErrNotExist here, by contract.
		return nil, fmt.Errorf("get(%q): %w", u.String(), os.ErrNotExist)
	default:
		return nil, fmt.Errorf("get(%q): %v", u.String(), r.StatusCode)
	}

	defer func() {
		if err := r.Body.Close(); err != nil {
			klog.Errorf("resp.Body.Close(): %v", err)
		}
	}()
	return io.ReadAll(r.Body)
}

func (h HTTPFetcher) ReadCheckpoint(ctx context.Context) ([]byte, error) {
	return h.fetch(ctx, layout.CheckpointPath)
}

func (h HTTPFetcher) ReadTile(ctx context.Context, l, i uint64, p uint8) ([]byte, error) {
	return fetcher.PartialOrFullResource(ctx, p, func(ctx context.Context, p uint8) ([]byte, error) {
		return h.fetch(ctx, layout.TilePath(l, i, p))
	})
}

func (h HTTPFetcher) ReadEntryBundle(ctx context.Context, i uint64, p uint8) ([]byte, error) {
	return fetcher.PartialOrFullResource(ctx, p, func(ctx context.Context, p uint8) ([]byte, error) {
		return h.fetch(ctx, layout.EntriesPath(i, p))
	})
}

// FileFetcher knows how to fetch log artifacts from a filesystem rooted at Root.
type FileFetcher struct {
	Root string
}

func (f FileFetcher) ReadCheckpoint(_ context.Context) ([]byte, error) {
	return os.ReadFile(path.Join(f.Root, layout.CheckpointPath))
}

func (f FileFetcher) ReadTile(ctx context.Context, l, i uint64, p uint8) ([]byte, error) {
	return fetcher.PartialOrFullResource(ctx, p, func(ctx context.Context, p uint8) ([]byte, error) {
		return os.ReadFile(path.Join(f.Root, layout.TilePath(l, i, p)))
	})
}

func (f FileFetcher) ReadEntryBundle(ctx context.Context, i uint64, p uint8) ([]byte, error) {
	return fetcher.PartialOrFullResource(ctx, p, func(ctx context.Context, p uint8) ([]byte, error) {
		return os.ReadFile(path.Join(f.Root, layout.EntriesPath(i, p)))
	})
}
