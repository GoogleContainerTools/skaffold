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

package timeutil

import (
	"context"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

func LessThan(date string, duration time.Duration) bool {
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		log.Entry(context.Background()).Debugf("could not parse date %q", date)
		return false
	}
	return time.Since(t) < duration
}
