/*
Copyright 2021 The Skaffold Authors

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

package time

import (
	"context"
	"strings"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

// LessThan returns true if the time since the given date is less than the given duration
func LessThan(date string, duration time.Duration) bool {
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		log.Entry(context.TODO()).Debugf("could not parse date %q", date)
		return false
	}
	return time.Since(t) < duration
}

// Humanize returns time in human readable format
func Humanize(start time.Duration) string {
	shortTime := start.Truncate(time.Millisecond)
	longTime := shortTime.String()
	out := time.Time{}.Add(shortTime)

	if start.Seconds() < 1 {
		return start.String()
	}

	longTime = strings.ReplaceAll(longTime, "h", " hour ")
	longTime = strings.ReplaceAll(longTime, "m", " minute ")
	longTime = strings.ReplaceAll(longTime, "s", " second")
	if out.Hour() > 1 {
		longTime = strings.ReplaceAll(longTime, "hour", "hours")
	}
	if out.Minute() > 1 {
		longTime = strings.ReplaceAll(longTime, "minute", "minutes")
	}
	if out.Second() > 1 {
		longTime = strings.ReplaceAll(longTime, "second", "seconds")
	}
	return longTime
}
