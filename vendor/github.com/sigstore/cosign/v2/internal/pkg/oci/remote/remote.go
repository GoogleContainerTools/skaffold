//
// Copyright 2023 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"fmt"
)

// ArtifactType converts a attachment name (sig/sbom/att/etc.) into a valid artifactType (OCI 1.1+).
func ArtifactType(attName string) string {
	return fmt.Sprintf("application/vnd.dev.cosign.artifact.%s.v1+json", attName)
}
