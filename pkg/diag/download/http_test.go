/*
Copyright 2020 The Skaffold Authors

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

package download

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestHTTPDownload(t *testing.T) {
	v, err := HTTPDownload("https://storage.googleapis.com/skaffold/releases/v1.0.0/VERSION")
	testutil.CheckError(t, false, err)
	testutil.CheckDeepEqual(t, "v1.0.0\n", string(v))
}
