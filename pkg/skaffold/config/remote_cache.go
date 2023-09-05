/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mitchellh/go-homedir"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
)

// These are the list of accepted values for flag `--sync-remote-cache`.
const (
	// clone missing remote dependencies and sync them on each skaffold run
	always = "always"
	// clone missing remote dependencies but do not sync found remote dependencies
	missing = "missing"
	// do not clone missing remote dependencies, and do not sync found remote dependencies
	never = "never"
)

// SyncRemoteCacheOption holds the value of flag `--sync-remote-cache`
// Valid flag values are `always`(default), `missing`, or `never`.
type SyncRemoteCacheOption struct {
	value string
}

func (s *SyncRemoteCacheOption) Type() string {
	return "string"
}

func (s *SyncRemoteCacheOption) Value() string {
	return s.value
}

func (s *SyncRemoteCacheOption) Set(v string) error {
	switch v {
	case always, missing, never:
		s.value = v
		return nil
	default:
		return errors.New("value must be one of `always`, `missing`, or `never`")
	}
}

func (s *SyncRemoteCacheOption) SetNil() error {
	s.value = always
	return nil
}

func (s *SyncRemoteCacheOption) String() string {
	if s.value == "" {
		return always
	}
	return s.value
}

// CloneDisabled specifies if cloning remote dependencies is disabled by flag value
func (s *SyncRemoteCacheOption) CloneDisabled() bool {
	return s.value == never
}

// FetchDisabled specifies if fetching remote dependencies is disabled by flag value
func (s *SyncRemoteCacheOption) FetchDisabled() bool {
	return s.value == missing || s.value == never
}

// GetRemoteCacheDir returns the directory for the remote cache.
func GetRemoteCacheDir(opts SkaffoldOptions) (string, error) {
	if opts.RemoteCacheDir != "" {
		return opts.RemoteCacheDir, nil
	}

	// cache location unspecified, use ~/.skaffold/remote-cache
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("retrieving home directory: %w", err)
	}
	return filepath.Join(home, constants.DefaultSkaffoldDir, "remote-cache"), nil
}
