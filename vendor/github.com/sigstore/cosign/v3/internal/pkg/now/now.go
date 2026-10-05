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

package now

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Now returns SOURCE_DATE_EPOCH or time.Now().
func Now() (time.Time, error) {
	// nolint
	epoch := os.Getenv("SOURCE_DATE_EPOCH")
	if epoch == "" {
		return time.Now(), nil
	}

	seconds, err := strconv.ParseInt(epoch, 10, 64)
	if err != nil {
		return time.Now(), fmt.Errorf("SOURCE_DATE_EPOCH should be the number of seconds since January 1st 1970, 00:00 UTC, got: %w", err)
	}
	return time.Unix(seconds, 0), nil
}
