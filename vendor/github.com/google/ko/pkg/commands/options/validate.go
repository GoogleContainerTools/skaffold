// Copyright 2022 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package options

import (
	"errors"
	"log"
	"strings"
)

const bareBaseFlagsWarning = `WARNING!
-----------------------------------------------------------------
Both --base-import-paths and --bare were set.

--base-import-paths will take precedence and ignore --bare flag.

In a future release this will be an error.
-----------------------------------------------------------------
`

const localFlagsWarning = `WARNING!
-----------------------------------------------------------------
The --local flag is set and KO_DOCKER_REPO is set to ko.local

You can choose either one to build a local image.

The --local flag might be deprecated in the future.
-----------------------------------------------------------------
`

func Validate(po *PublishOptions, bo *BuildOptions) error {
	po.Jobs = bo.ConcurrentBuilds
	if po.Bare && po.BaseImportPaths {
		log.Print(bareBaseFlagsWarning)
		// TODO: return error when we decided to make this an error, for now it is a warning
	}

	if po.Local && strings.Contains(po.DockerRepo, "ko.local") {
		log.Print(localFlagsWarning)
	}

	if len(bo.Platforms) > 1 {
		for _, platform := range bo.Platforms {
			if platform == "all" {
				return errors.New("all or specific platforms should be used")
			}
		}
	}

	return nil
}
