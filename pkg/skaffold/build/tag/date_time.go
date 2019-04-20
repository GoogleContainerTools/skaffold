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
	"fmt"
	"time"

	"4d63.com/tz"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

const tagTime = "2006-01-02_15-04-05.999_MST"

// dateTimeTagger tags an image by the timestamp of the built image
// dateTimeTagger implements Tagger
type dateTimeTagger struct {
	Format   string
	TimeZone string
	timeFn   func() time.Time
}

// NewDateTimeTagger creates a tagger from a date format and timezone.
func NewDateTimeTagger(format, timezone string) Tagger {
	return &dateTimeTagger{
		Format:   format,
		TimeZone: timezone,
		timeFn:   time.Now,
	}
}

func (tagger *dateTimeTagger) Labels() map[string]string {
	return map[string]string{
		constants.Labels.TagPolicy: "dateTimeTagger",
	}
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the current timestamp
func (tagger *dateTimeTagger) GenerateFullyQualifiedImageName(workingDir, imageName string) (string, error) {
	format := tagTime
	if len(tagger.Format) > 0 {
		format = tagger.Format
	}

	timezone := "Local"
	if len(tagger.TimeZone) > 0 {
		timezone = tagger.TimeZone
	}

	loc, err := tz.LoadLocation(timezone)
	if err != nil {
		return "", fmt.Errorf("bad timezone provided: \"%s\", error: %s", timezone, err)
	}

	return fmt.Sprintf("%s:%s", imageName, tagger.timeFn().In(loc).Format(format)), nil
}
