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

package tag

import (
	"fmt"

	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

const tagTime = "2006-01-02_15-04-05.999_MST"

// dateTimeTagger tags an image by the timestamp of the built image
// dateTimeTagger implements Tagger
type dateTimeTagger struct {
	Format   string
	TimeZone string
	Clock    util.Clock
}

func NewDateTimeTagger(format, timezone string) (*dateTimeTagger, error) {
	return &dateTimeTagger{
		Format:   format,
		TimeZone: timezone,
		Clock:    &util.RealClock{},
	}, nil
}

// GenerateFullyQualifiedImageName tags an image with the supplied image name and the current timestamp
func (tagger *dateTimeTagger) GenerateFullyQualifiedImageName(workingDir string, opts *TagOptions) (string, error) {
	if opts == nil {
		return "", fmt.Errorf("tag options not provided")
	}

	c := tagger.Clock

	format := tagTime
	if len(tagger.Format) > 0 {
		format = tagger.Format
	}

	timezone := "Local"
	if len(tagger.TimeZone) > 0 {
		timezone = tagger.TimeZone
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return "", fmt.Errorf("bad timezone provided: \"%s\", error: %s", timezone, err)
	}

	return opts.ImageName + ":" + c.Now().In(loc).Format(format), nil
}
