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

package tag

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateFullyQualifiedImageName(t *testing.T) {
	c := &ChecksumTagger{}
	var checksum []byte

	//Creates temporary directory containing 1 file.
	wd, cleanUp := testutil.NewTempDir(t)
	defer cleanUp()

	wd.Write("test1", "code")
	checksum = computeTmpChecksum(wd.Path("test1"), checksum)

	tag, err := c.GenerateFullyQualifiedImageName(wd.Root(), "img")
	testutil.CheckErrorAndDeepEqual(t, false, err, fmt.Sprintf("img:%x", md5.Sum(checksum)), tag)

	tag, err = c.GenerateFullyQualifiedImageName(wd.Root(), "registry.example.com:8080/img")
	testutil.CheckErrorAndDeepEqual(t, false, err, fmt.Sprintf("registry.example.com:8080/img:%x", md5.Sum(checksum)), tag)

	tag, err = c.GenerateFullyQualifiedImageName(wd.Root(), "registry.example.com/img")
	testutil.CheckErrorAndDeepEqual(t, false, err, fmt.Sprintf("registry.example.com/img:%x", md5.Sum(checksum)), tag)

	//Add 1 file to the already created directory
	wd.Write("test2", "code in a new file")
	checksum = computeTmpChecksum(wd.Path("test2"), checksum)
	tag, err = c.GenerateFullyQualifiedImageName(wd.Root(), "img")
	testutil.CheckErrorAndDeepEqual(t, false, err, fmt.Sprintf("img:%x", md5.Sum(checksum)), tag)

	tag, err = c.GenerateFullyQualifiedImageName(".", "img:tag")
	testutil.CheckErrorAndDeepEqual(t, false, err, "img:tag", tag)

	tag, err = c.GenerateFullyQualifiedImageName(".", "registry.example.com:8080/img:tag")
	testutil.CheckErrorAndDeepEqual(t, false, err, "registry.example.com:8080/img:tag", tag)

	tag, err = c.GenerateFullyQualifiedImageName(".", "registry.example.com:8080:garbage")
	testutil.CheckErrorAndDeepEqual(t, true, err, "", tag)
}

func computeTmpChecksum(path string, checksum []byte) []byte {
	f, _ := os.Open(path)
	defer f.Close()

	h := sha256.New()
	io.Copy(h, f)
	return h.Sum(checksum)
}
