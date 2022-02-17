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

package lsp

import (
	"context"
	"net/url"
	"strings"

	"go.lsp.dev/uri"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

const gitPrefix = "git:/"

func uriToFilename(v uri.URI) string {
	s := string(v)
	fixed, ok := fixURI(s)

	if !ok {
		unescaped, err := url.PathUnescape(s)
		if err != nil {
			log.Entry(context.TODO()).Warnf("Unsupported URI (not a filepath): %q\n", s)
		} else {
			log.Entry(context.TODO()).Warnf("Unsupported URI (not a filepath): %q\n", unescaped)
		}
		return ""
	}
	v = uri.URI(fixed)

	return v.Filename()
}

// workaround for unsupported file paths (git + invalid file://-prefix )
func fixURI(s string) (string, bool) {
	if strings.HasPrefix(s, gitPrefix) {
		return "file:///" + s[len(gitPrefix):], true
	}
	if !strings.HasPrefix(s, "file:///") {
		// VS Code sends URLs with only two slashes, which are invalid. golang/go#39789.
		if strings.HasPrefix(s, "file://") {
			return "file:///" + s[len("file://"):], true
		}
		return "", false
	}
	return s, true
}
