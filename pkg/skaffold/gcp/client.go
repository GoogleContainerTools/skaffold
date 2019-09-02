/*
Copyright 2019 The Skaffold Authors

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

package gcp

import (
	"context"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"

	cstorage "cloud.google.com/go/storage"
	"github.com/pkg/errors"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/option"
)

var (
	cbclient *cloudbuild.Service
	cbOnce   sync.Once
	cbErr    error
)

// CloudBuildClient returns an authenticated client for interacting with
// the Google Cloud Build API. This client is created once and cached
// for repeated use in Skaffold.
func CloudBuildClient() (*cloudbuild.Service, error) {
	cbOnce.Do(func() {
		var options []option.ClientOption
		var err error
		creds, cErr := activeUserCredentials()
		if cErr == nil && creds != nil {
			options = append(options, option.WithCredentials(creds))
		}

		c, err := cloudbuild.NewService(context.Background(), options...)
		if err != nil {
			cbErr = err
			return
		}
		c.UserAgent = version.UserAgent()
		cbclient = c
	})

	return cbclient, cbErr
}

// CloudStorageClient returns an authenticated client for interacting with
// the Google Cloud Storage API. This client is not cached by Skaffold,
// because it needs to be closed each time it is done being used by the caller.
func CloudStorageClient() (*cstorage.Client, error) {
	var options []option.ClientOption
	var err error
	creds, cErr := activeUserCredentials()
	if cErr == nil && creds != nil {
		options = append(options, option.WithCredentials(creds))
	}
	c, err := cstorage.NewClient(context.Background(), options...)
	if err != nil {
		return nil, errors.Wrap(err, "getting cloud storage client")
	}
	return c, nil
}
