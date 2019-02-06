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

import (
	"errors"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

type CustomTag struct {
	Tag string
}

func (c *CustomTag) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "custom",
	}
}

// GenerateFullyQualifiedImageName tags an image with the custom tag
func (c *CustomTag) GenerateFullyQualifiedImageName(workingDir, imageName string) (string, error) {
	tag := c.Tag
	if tag == "" {
		return "", errors.New("custom tag not provided")
	}

	return fmt.Sprintf("%s:%s", imageName, tag), nil
}
