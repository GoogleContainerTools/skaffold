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
	"google.golang.org/api/option"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

// ClientOptions returns a list of options to be configured when
// connecting to Google Cloud services.
func ClientOptions() []option.ClientOption {
	options := []option.ClientOption{
		option.WithUserAgent(version.UserAgent()),
	}

	creds, cErr := activeUserCredentials()
	if cErr == nil && creds != nil {
		options = append(options, option.WithCredentials(creds))
	}

	return options
}
