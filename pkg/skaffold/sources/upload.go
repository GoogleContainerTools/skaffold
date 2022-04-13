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

package sources

import (
	"context"
	"fmt"

	cstorage "cloud.google.com/go/storage"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// UploadToGCS uploads the artifact's sources to a GCS bucket.
func UploadToGCS(ctx context.Context, c *cstorage.Client, a *latest.Artifact, bucket, objectName string, dependencies []string) error {
	w := c.Bucket(bucket).Object(objectName).NewWriter(ctx)

	if err := util.CreateTarGz(w, a.Workspace, dependencies); err != nil {
		return fmt.Errorf("uploading sources to google storage: %w", err)
	}

	return w.Close()
}
