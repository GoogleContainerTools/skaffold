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

package util

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDownload_UserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		w.Write([]byte(ua))
	}))
	defer ts.Close()

	// although we don't have a version number in tests, the user-agent string still
	// has `skaffold/<GOOS>/<GOARCH>/`
	testutil.CheckDeepEqual(t, true, strings.HasPrefix(version.UserAgent(), "skaffold/"))

	v, err := Download(ts.URL)
	testutil.CheckErrorAndDeepEqual(t, false, err, version.UserAgent(), string(v))
}
