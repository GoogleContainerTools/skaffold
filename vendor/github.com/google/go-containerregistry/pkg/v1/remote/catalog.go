// Copyright 2019 Google LLC All Rights Reserved.
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

package remote

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

type catalog struct {
	Repos []string `json:"repositories"`
}

// CatalogPage calls /_catalog, returning the list of repositories on the registry.
func CatalogPage(target name.Registry, last string, n int, options ...Option) ([]string, error) {
	o, err := makeOptions(target, options...)
	if err != nil {
		return nil, err
	}

	scopes := []string{target.Scope(transport.PullScope)}
	tr, err := transport.New(target, o.auth, o.transport, scopes)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("last=%s&n=%d", url.QueryEscape(last), n)

	uri := url.URL{
		Scheme:   target.Scheme(),
		Host:     target.RegistryStr(),
		Path:     "/v2/_catalog",
		RawQuery: query,
	}

	client := http.Client{Transport: tr}
	resp, err := client.Get(uri.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := transport.CheckError(resp, http.StatusOK); err != nil {
		return nil, err
	}

	var parsed catalog
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	return parsed.Repos, nil
}
