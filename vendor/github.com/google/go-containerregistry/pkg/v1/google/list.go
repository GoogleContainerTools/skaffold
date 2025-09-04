// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// Option is a functional option for List and Walk.
// TODO: Can we somehow reuse the remote options here?
type Option func(*lister) error

type lister struct {
	auth      authn.Authenticator
	transport http.RoundTripper
	repo      name.Repository
	client    *http.Client
	ctx       context.Context
	userAgent string
}

func newLister(repo name.Repository, options ...Option) (*lister, error) {
	l := &lister{
		auth:      authn.Anonymous,
		transport: http.DefaultTransport,
		repo:      repo,
		ctx:       context.Background(),
	}

	for _, option := range options {
		if err := option(l); err != nil {
			return nil, err
		}
	}

	// transport.Wrapper is a signal that consumers are opt-ing into providing their own transport without any additional wrapping.
	// This is to allow consumers full control over the transports logic, such as providing retry logic.
	if _, ok := l.transport.(*transport.Wrapper); !ok {
		// Wrap the transport in something that logs requests and responses.
		// It's expensive to generate the dumps, so skip it if we're writing
		// to nothing.
		if logs.Enabled(logs.Debug) {
			l.transport = transport.NewLogger(l.transport)
		}

		// Wrap the transport in something that can retry network flakes.
		l.transport = transport.NewRetry(l.transport)

		// Wrap this last to prevent transport.New from double-wrapping.
		if l.userAgent != "" {
			l.transport = transport.NewUserAgent(l.transport, l.userAgent)
		}
	}

	scopes := []string{repo.Scope(transport.PullScope)}
	tr, err := transport.NewWithContext(l.ctx, repo.Registry, l.auth, l.transport, scopes)
	if err != nil {
		return nil, err
	}

	l.client = &http.Client{Transport: tr}

	return l, nil
}

func (l *lister) list(repo name.Repository) (*Tags, error) {
	uri := &url.URL{
		Scheme:   repo.Scheme(),
		Host:     repo.RegistryStr(),
		Path:     fmt.Sprintf("/v2/%s/tags/list", repo.RepositoryStr()),
		RawQuery: "n=10000",
	}

	// ECR returns an error if n > 1000:
	// https://github.com/google/go-containerregistry/issues/681
	if !isGoogle(repo.RegistryStr()) {
		uri.RawQuery = "n=1000"
	}

	tags := Tags{}

	// get responses until there is no next page
	for {
		select {
		case <-l.ctx.Done():
			return nil, l.ctx.Err()
		default:
		}

		req, err := http.NewRequest("GET", uri.String(), nil)
		if err != nil {
			return nil, err
		}
		req = req.WithContext(l.ctx)

		resp, err := l.client.Do(req)
		if err != nil {
			return nil, err
		}

		if err := transport.CheckError(resp, http.StatusOK); err != nil {
			return nil, err
		}

		parsed := Tags{}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return nil, err
		}

		if err := resp.Body.Close(); err != nil {
			return nil, err
		}

		if len(parsed.Manifests) != 0 || len(parsed.Children) != 0 {
			// We're dealing with GCR, just return directly.
			return &parsed, nil
		}

		// This isn't GCR, just append the tags and keep paginating.
		tags.Tags = append(tags.Tags, parsed.Tags...)

		uri, err = getNextPageURL(resp)
		if err != nil {
			return nil, err
		}
		// no next page
		if uri == nil {
			break
		}
		logs.Warn.Printf("saw non-google tag listing response, falling back to pagination")
	}

	return &tags, nil
}

// getNextPageURL checks if there is a Link header in a http.Response which
// contains a link to the next page. If yes it returns the url.URL of the next
// page otherwise it returns nil.
func getNextPageURL(resp *http.Response) (*url.URL, error) {
	link := resp.Header.Get("Link")
	if link == "" {
		return nil, nil
	}

	if link[0] != '<' {
		return nil, fmt.Errorf("failed to parse link header: missing '<' in: %s", link)
	}

	end := strings.Index(link, ">")
	if end == -1 {
		return nil, fmt.Errorf("failed to parse link header: missing '>' in: %s", link)
	}
	link = link[1:end]

	linkURL, err := url.Parse(link)
	if err != nil {
		return nil, err
	}
	if resp.Request == nil || resp.Request.URL == nil {
		return nil, nil
	}
	linkURL = resp.Request.URL.ResolveReference(linkURL)
	return linkURL, nil
}

type rawManifestInfo struct {
	Size      string   `json:"imageSizeBytes"`
	MediaType string   `json:"mediaType"`
	Created   string   `json:"timeCreatedMs"`
	Uploaded  string   `json:"timeUploadedMs"`
	Tags      []string `json:"tag"`
}

// ManifestInfo is a Manifests entry is the output of List and Walk.
type ManifestInfo struct {
	Size      uint64    `json:"imageSizeBytes"`
	MediaType string    `json:"mediaType"`
	Created   time.Time `json:"timeCreatedMs"`
	Uploaded  time.Time `json:"timeUploadedMs"`
	Tags      []string  `json:"tag"`
}

func fromUnixMs(ms int64) time.Time {
	sec := ms / 1000
	ns := (ms % 1000) * 1000000
	return time.Unix(sec, ns)
}

func toUnixMs(t time.Time) string {
	return strconv.FormatInt(t.UnixNano()/1000000, 10)
}

// MarshalJSON implements json.Marshaler
func (m ManifestInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(rawManifestInfo{
		Size:      strconv.FormatUint(m.Size, 10),
		MediaType: m.MediaType,
		Created:   toUnixMs(m.Created),
		Uploaded:  toUnixMs(m.Uploaded),
		Tags:      m.Tags,
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (m *ManifestInfo) UnmarshalJSON(data []byte) error {
	raw := rawManifestInfo{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.Size != "" {
		size, err := strconv.ParseUint(raw.Size, 10, 64)
		if err != nil {
			return err
		}
		m.Size = size
	}

	if raw.Created != "" {
		created, err := strconv.ParseInt(raw.Created, 10, 64)
		if err != nil {
			return err
		}
		m.Created = fromUnixMs(created)
	}

	if raw.Uploaded != "" {
		uploaded, err := strconv.ParseInt(raw.Uploaded, 10, 64)
		if err != nil {
			return err
		}
		m.Uploaded = fromUnixMs(uploaded)
	}

	m.MediaType = raw.MediaType
	m.Tags = raw.Tags

	return nil
}

// Tags is the result of List and Walk.
type Tags struct {
	Children  []string                `json:"child"`
	Manifests map[string]ManifestInfo `json:"manifest"`
	Name      string                  `json:"name"`
	Tags      []string                `json:"tags"`
}

// List calls /tags/list for the given repository.
func List(repo name.Repository, options ...Option) (*Tags, error) {
	l, err := newLister(repo, options...)
	if err != nil {
		return nil, err
	}

	return l.list(repo)
}

// WalkFunc is the type of the function called for each repository visited by
// Walk. This implements a similar API to filepath.Walk.
//
// The repo argument contains the argument to Walk as a prefix; that is, if Walk
// is called with "gcr.io/foo", which is a repository containing the repository
// "bar", the walk function will be called with argument "gcr.io/foo/bar".
// The tags and error arguments are the result of calling List on repo.
//
// TODO: Do we want a SkipDir error, as in filepath.WalkFunc?
type WalkFunc func(repo name.Repository, tags *Tags, err error) error

func walk(repo name.Repository, tags *Tags, walkFn WalkFunc, options ...Option) error {
	if tags == nil {
		// This shouldn't happen.
		return fmt.Errorf("tags nil for %q", repo)
	}

	if err := walkFn(repo, tags, nil); err != nil {
		return err
	}

	for _, path := range tags.Children {
		child, err := name.NewRepository(fmt.Sprintf("%s/%s", repo, path), name.StrictValidation)
		if err != nil {
			// We don't expect this ever, so don't pass it through to walkFn.
			return fmt.Errorf("unexpected path failure: %w", err)
		}

		childTags, err := List(child, options...)
		if err != nil {
			if err := walkFn(child, nil, err); err != nil {
				return err
			}
		} else {
			if err := walk(child, childTags, walkFn, options...); err != nil {
				return err
			}
		}
	}

	// We made it!
	return nil
}

// Walk recursively descends repositories, calling walkFn.
func Walk(root name.Repository, walkFn WalkFunc, options ...Option) error {
	tags, err := List(root, options...)
	if err != nil {
		return walkFn(root, nil, err)
	}

	return walk(root, tags, walkFn, options...)
}
