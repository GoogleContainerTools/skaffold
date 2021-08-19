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

package user

import (
	"fmt"
	"regexp"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
)

func IsAllowedUser(user string) bool {
	for allowedUser := range constants.AllowedUsers {
		matched, err := regexp.MatchString(fmt.Sprintf(constants.AllowedUserPattern, allowedUser), user)
		if err != nil {
			panic(fmt.Sprintf("error matching allowed user: %v", err))
		}

		if matched {
			return true
		}
	}

	return false
}
