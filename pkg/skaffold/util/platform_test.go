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

package util

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestConvert(t *testing.T) {
	pl1 := v1.Platform{Architecture: "arch", OS: "os", OSVersion: "os_ver", Variant: "variant", OSFeatures: []string{"os_feature"}}
	pl2 := ConvertFromV1Platform(pl1)
	pl3 := ConvertToV1Platform(pl2)
	testutil.CheckDeepEqual(t, pl1, pl3)
}
