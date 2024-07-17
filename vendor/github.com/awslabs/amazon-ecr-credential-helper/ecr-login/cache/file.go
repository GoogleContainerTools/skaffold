// Copyright 2016 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const registryCacheVersion = "1.0"

type RegistryCache struct {
	Registries map[string]*AuthEntry
	Version    string
}

type fileCredentialCache struct {
	path           string
	filename       string
	cachePrefixKey string
	publicCacheKey string
}

func newRegistryCache() *RegistryCache {
	return &RegistryCache{
		Registries: make(map[string]*AuthEntry),
		Version:    registryCacheVersion,
	}
}

// NewFileCredentialsCache returns a new file credentials cache.
//
// path is used for temporary files during save, and filename should be a relative filename
// in the same directory where the cache is serialized and deserialized.
//
// cachePrefixKey is used for scoping credentials for a given credential cache (i.e. region and
// accessKey).
func NewFileCredentialsCache(path string, filename string, cachePrefixKey string, publicCacheKey string) CredentialsCache {
	if _, err := os.Stat(path); err != nil {
		os.MkdirAll(path, 0700)
	}
	return &fileCredentialCache{
		path:           path,
		filename:       filename,
		cachePrefixKey: cachePrefixKey,
		publicCacheKey: publicCacheKey,
	}
}

func (f *fileCredentialCache) Get(registry string) *AuthEntry {
	logrus.WithField("registry", registry).Debug("Checking file cache")
	registryCache := f.init()
	return registryCache.Registries[f.cachePrefixKey+registry]
}

func (f *fileCredentialCache) GetPublic() *AuthEntry {
	logrus.Debug("Checking file cache for ECR Public")
	registryCache := f.init()
	return registryCache.Registries[f.publicCacheKey]
}

func (f *fileCredentialCache) Set(registry string, entry *AuthEntry) {
	logrus.
		WithField("registry", registry).
		WithField("service", entry.Service).
		Debug("Saving credentials to file cache")
	registryCache := f.init()

	key := f.cachePrefixKey + registry
	if entry.Service == ServiceECRPublic {
		key = f.publicCacheKey
	}
	registryCache.Registries[key] = entry

	err := f.save(registryCache)
	if err != nil {
		logrus.WithError(err).Info("Could not save cache")
	}
}

// List returns all of the available AuthEntries (regardless of prefix)
func (f *fileCredentialCache) List() []*AuthEntry {
	registryCache := f.init()

	// optimize allocation for copy
	entries := make([]*AuthEntry, 0, len(registryCache.Registries))

	for _, entry := range registryCache.Registries {
		entries = append(entries, entry)
	}

	return entries
}

func (f *fileCredentialCache) Clear() {
	err := os.Remove(f.fullFilePath())
	if err != nil {
		logrus.WithError(err).Info("Could not clear cache")
	}
}

func (f *fileCredentialCache) fullFilePath() string {
	return filepath.Join(f.path, f.filename)
}

// Saves credential cache to disk. This writes to a temporary file first, then moves the file to the config location.
// This eliminates from reading partially written credential files, and reduces (but does not eliminate) concurrent
// file access. There is not guarantee here for handling multiple writes at once since there is no out of process locking.
func (f *fileCredentialCache) save(registryCache *RegistryCache) error {
	file, err := os.CreateTemp(f.path, ".config.json.tmp")
	if err != nil {
		return err
	}

	buff, err := json.MarshalIndent(registryCache, "", "  ")
	if err != nil {
		file.Close()
		os.Remove(file.Name())
		return err
	}

	_, err = file.Write(buff)

	if err != nil {
		file.Close()
		os.Remove(file.Name())
		return err
	}

	file.Close()
	// note this is only atomic when relying on linux syscalls
	os.Rename(file.Name(), f.fullFilePath())
	return err
}

func (f *fileCredentialCache) init() *RegistryCache {
	registryCache, err := f.load()
	if err != nil {
		logrus.WithError(err).Info("Could not load existing cache")
		f.Clear()
		registryCache = newRegistryCache()
	}
	return registryCache
}

// Loading a cache from disk will return errors for malformed or incompatible cache files.
func (f *fileCredentialCache) load() (*RegistryCache, error) {
	registryCache := newRegistryCache()

	file, err := os.Open(f.fullFilePath())
	if os.IsNotExist(err) {
		return registryCache, nil
	}

	if err != nil {
		return nil, err
	}

	defer file.Close()

	if err = json.NewDecoder(file).Decode(&registryCache); err != nil {
		return nil, err
	}

	if registryCache.Version != registryCacheVersion {
		return nil, fmt.Errorf("ecr: Registry cache version %#v is not compatible with %#v, ignoring existing cache",
			registryCache.Version,
			registryCacheVersion)
	}

	// migrate entries
	for key := range registryCache.Registries {
		if registryCache.Registries[key].Service == "" {
			registryCache.Registries[key].Service = ServiceECR
		}
	}

	return registryCache, nil
}
