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

package tag

// ImageTags maps image names to tags
type ImageTags map[string]string

// Tagger is an interface for tag strategies to be implemented against
type Tagger interface {
	// Labels produces labels to indicate the used tagger in deployed pods.
	Labels() map[string]string

	// GenerateFullyQualifiedImageName resolves the fully qualified image name for an artifact.
	// The workingDir is the root directory of the artifact with respect to the Skaffold root,
	// and imageName is the base name of the image.
	GenerateFullyQualifiedImageName(workingDir string, imageName string) (string, error)
}
