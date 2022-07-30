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

package schema

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestListPlain(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var out bytes.Buffer

		err := list(&out, "plain")
		t.CheckNoError(err)

		versions := out.String()
		t.CheckTrue(strings.HasSuffix(versions, latest.Version+"\n"))
		t.CheckTrue(strings.HasPrefix(versions, `skaffold/v1alpha1
skaffold/v1alpha2
skaffold/v1alpha3
skaffold/v1alpha4
skaffold/v1alpha5
skaffold/v1beta1
skaffold/v1beta2
skaffold/v1beta3
skaffold/v1beta4
skaffold/v1beta5
skaffold/v1beta6
skaffold/v1beta7
skaffold/v1beta8
skaffold/v1beta9
skaffold/v1beta10
skaffold/v1beta11
skaffold/v1beta12
skaffold/v1beta13
skaffold/v1beta14
skaffold/v1beta15
skaffold/v1beta16
skaffold/v1beta17
skaffold/v1
skaffold/v2alpha1`))
	})
}

func TestListJson(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var out bytes.Buffer

		err := list(&out, "json")
		t.CheckNoError(err)

		versions := out.String()
		t.CheckTrue(strings.HasPrefix(versions, `{"versions":["skaffold/v1alpha1","skaffold/v1alpha2",`))
		t.CheckTrue(strings.HasSuffix(versions, fmt.Sprintf(",\"%s\"]}\n", latest.Version)))
	})
}

func TestListInvalidType(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var out bytes.Buffer

		err := list(&out, "invalid")
		t.CheckErrorContains(`invalid output type: "invalid". Must be "plain" or "json"`, err)
	})
}
