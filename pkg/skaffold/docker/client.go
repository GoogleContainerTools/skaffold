/*
Copyright 2018 Google LLC

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

package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/moby/moby/client"
)

// NewImageAPIClient returns a docker client based on the environment variables set.
// It will "negotiate" the highest possible API version supported by both the client
// and the server if there is a mismatch.
func NewImageAPIClient() (client.ImageAPIClient, io.Closer, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, nil, fmt.Errorf("Error getting docker client: %s", err)
	}
	cli.NegotiateAPIVersion(context.Background())

	return cli, cli, nil
}
