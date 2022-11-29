/*
Copyright 2022 The Skaffold Authors

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

package manifest

import (
	"errors"
	"testing"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/proto/enums"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestReplaceImageErr(t *testing.T) {
	testutil.Run(t, "TestReplaceImageErr", func(t *testutil.T) {
		err := replaceImageErr(errors.New(""))
		t.CheckDeepEqual(err.(*sErrors.ErrDef).StatusCode(), enums.StatusCode_RENDER_REPLACE_IMAGE_ERR)
	})
}

func TestTransformManifestErr(t *testing.T) {
	testutil.Run(t, "TestReplaceImageErr", func(t *testutil.T) {
		err := transformManifestErr(errors.New(""))
		t.CheckDeepEqual(err.(*sErrors.ErrDef).StatusCode(), enums.StatusCode_RENDER_TRANSFORM_MANIFEST_ERR)
	})
}

func TestLabelSettingErr(t *testing.T) {
	testutil.Run(t, "TestLabelSettingErr", func(t *testutil.T) {
		err := labelSettingErr(errors.New(""))
		t.CheckDeepEqual(err.(*sErrors.ErrDef).StatusCode(), enums.StatusCode_RENDER_SET_LABEL_ERR)
	})
}

func TestParseImagesInManifestErr(t *testing.T) {
	testutil.Run(t, "TestParseImagesInManifestErr", func(t *testutil.T) {
		err := parseImagesInManifestErr(errors.New(""))
		t.CheckDeepEqual(err.(*sErrors.ErrDef).StatusCode(), enums.StatusCode_RENDER_PARSE_MANIFEST_IMAGES_ERR)
	})
}

func TestWriteErr(t *testing.T) {
	testutil.Run(t, "TestWriteErr", func(t *testutil.T) {
		err := writeErr(errors.New(""))
		t.CheckDeepEqual(err.(*sErrors.ErrDef).StatusCode(), enums.StatusCode_RENDER_MANIFEST_WRITE_ERR)
	})
}

func TestNSSettingErr(t *testing.T) {
	testutil.Run(t, "TestNSSettingErr", func(t *testutil.T) {
		err := nsSettingErr(errors.New(""))
		t.CheckDeepEqual(err.(*sErrors.ErrDef).StatusCode(), enums.StatusCode_RENDER_SET_NAMESPACE_ERR)
	})
}
