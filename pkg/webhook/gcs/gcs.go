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

package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	cstorage "cloud.google.com/go/storage"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/GoogleContainerTools/skaffold/pkg/webhook/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/webhook/kubernetes"
)

// UploadDeploymentLogsToBucket gets logs from d and uploads them to the bucket
func UploadDeploymentLogsToBucket(d *appsv1.Deployment, prNumber int) (string, error) {
	c, err := cstorage.NewClient(context.Background())
	if err != nil {
		return "", fmt.Errorf("creating GCS client: %w", err)
	}
	defer c.Close()
	name := fmt.Sprintf("logs-%d-%d", prNumber, time.Now().UnixNano())
	w := c.Bucket(constants.LogsGCSBucket).Object(name).NewWriter(context.Background())
	defer w.Close()
	if _, err := io.Copy(w, bytes.NewBuffer([]byte(kubernetes.Logs(d)))); err != nil {
		return "", err
	}
	return name, nil
}
