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

package structure

import (
	"fmt"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

func containerStructureTestErr(err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: fmt.Sprintf("running container-structure-test: %s", err),
			ErrCode: proto.StatusCode_TEST_CST_USER_ERR,
		},
	)
}

func expandingFilePathsErr(err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: fmt.Sprintf("expanding test file paths: %s", err),
			ErrCode: proto.StatusCode_TEST_USER_CONFIG_ERR,
		},
	)
}

func dockerPullImageErr(fqn string, err error) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: fmt.Sprintf("unable to docker pull image %s: %s", fqn, err),
			ErrCode: proto.StatusCode_TEST_IMG_PULL_ERR,
		},
	)
}
