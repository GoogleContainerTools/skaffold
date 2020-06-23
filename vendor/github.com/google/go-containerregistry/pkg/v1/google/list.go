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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// ListerOption is a functional option for List and Walk.
// TODO: Can we somehow reuse the remote options here?
type ListerOption func(*lister) error

type lister struct {
	auth      authn.Authenticator
	transport http.RoundTripper
	repo      name.Repository
	client    *http.Client
}

func newLister(repo name.Repository, options ...ListerOption) (*lister, error) {
	l := &lister{
		auth:      authn.Anonymous,
		transport: http.DefaultTransport,
		repo:      repo,
	}

	for _, option := range options {
		if err := option(l); err != nil {
			return nil, err
		}
	}

	// Wrap the transport in something that logs requests and responses.
	// It's expensive to generate the dumps, so skip it if we're writing
	// to nothing.
	if logs.Enabled(logs.Debug) {
		l.transport = transport.NewLogger(l.transport)
	}

	// Wrap the transport in something that can retry network flakes.
	l.transport = transport.NewRetry(l.transport)

	scopes := []string{repo.Scope(transport.PullScope)}
	tr, err := transport.New(repo.Registry, l.auth, l.transport, scopes)
	if err != nil {
		return nil, err
	}

	l.client = &http.Client{Transport: tr}

	return l, nil
}

func (l *lister) list(repo name.Repository) (*Tags, error) {
	uri := url.URL{
		Scheme: repo.Registry.Scheme(),
		Host:   repo.Registry.RegistryStr(),
		Path:   fmt.Sprintf("/v2/%s/tags/list", repo.RepositoryStr()),
	}

	resp, err := l.client.Get(uri.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := transport.CheckError(resp, http.StatusOK); err != nil {
		return nil, err
	}

	tags := Tags{}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}

	return &tags, nil
}

// Uploaded uses json.Number to work around GCR returning timeUploaded
// as a string, while AR returns the same field as int64.
type rawManifestInfo struct {
	Size      string      `json:"imageSizeBytes"`
	MediaType string      `json:"mediaType"`
	Created   string      `json:"timeCreatedMs"`
	Uploaded  json.Number `json:"timeUploadedMs"`
	Tags      []string    `json:"tag"`
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
		Uploaded:  json.Number(toUnixMs(m.Uploaded)),
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
		size, err := strconv.ParseUint(string(raw.Size), 10, 64)
		if err != nil {
			return err
		}
		m.Size = size
	}

	if raw.Created != "" {
		created, err := strconv.ParseInt(string(raw.Created), 10, 64)
		if err != nil {
			return err
		}
		m.Created = fromUnixMs(created)
	}

	if raw.Uploaded != "" {
		uploaded, err := strconv.ParseInt(string(raw.Uploaded), 10, 64)
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
func List(repo name.Repository, options ...ListerOption) (*Tags, error) {
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

func walk(repo name.Repository, tags *Tags, walkFn WalkFunc, options ...ListerOption) error {
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
			return fmt.Errorf("unexpected path failure: %v", err)
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
func Walk(root name.Repository, walkFn WalkFunc, options ...ListerOption) error {
	tags, err := List(root, options...)
	if err != nil {
		return walkFn(root, nil, err)
	}

	return walk(root, tags, walkFn, options...)
}
