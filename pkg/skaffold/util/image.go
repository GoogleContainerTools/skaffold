/*
Copyright 2018 The Skaffold Authors

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

package util

import (
	"strings"
)

func SubstituteDefaultRepoIntoImage(defaultRepo string, originalImage string) string {
	if defaultRepo == "" {
		return originalImage
	}
	if strings.HasPrefix(defaultRepo, "gcr.io") {
		if !strings.HasPrefix(originalImage, defaultRepo) {
			return defaultRepo + "/" + originalImage
		} else {
			// TODO: this one is a little harder
			return originalImage
		}
	} else {
		// TODO: escape, concat, truncate to 256
		return originalImage
	}
	return originalImage
}
