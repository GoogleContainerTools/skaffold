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

package cache

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"sort"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

var (
	// For testing
	hashFunction = cacheHasher
)

func getHashForArtifact(ctx context.Context, builder build.Builder, a *latest.Artifact) (string, error) {
	deps, err := builder.DependenciesForArtifact(ctx, a)
	if err != nil {
		return "", errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
	}
	sort.Strings(deps)
	var hashes []string
	for _, d := range deps {
		h, err := hashFunction(d)
		if err != nil {
			return "", errors.Wrapf(err, "getting hash for %s", d)
		}
		hashes = append(hashes, h)
	}
	// get a key for the hashes
	c := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(c)
	enc.Encode(hashes)
	return util.SHA256(c)
}

// cacheHasher takes hashes the contents and name of a file
func cacheHasher(p string) (string, error) {
	h := md5.New()
	fi, err := os.Lstat(p)
	if err != nil {
		return "", err
	}
	h.Write([]byte(fi.Mode().String()))
	h.Write([]byte(fi.Name()))
	if fi.Mode().IsRegular() {
		f, err := os.Open(p)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
