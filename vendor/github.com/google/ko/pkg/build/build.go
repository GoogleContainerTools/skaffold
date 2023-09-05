// Copyright 2018 ko Build Authors All Rights Reserved.
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

package build

import (
	"context"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// Interface abstracts different methods for turning a supported importpath
// reference into a v1.Image.
type Interface interface {
	// QualifyImport turns relative importpath references into complete importpaths.
	// It also adds the ko scheme prefix if necessary.
	// E.g., "github.com/ko-build/ko/test" => "ko://github.com/ko-build/ko/test"
	// and "./test" => "ko://github.com/ko-build/ko/test"
	QualifyImport(string) (string, error)

	// IsSupportedReference determines whether the given reference is to an
	// importpath reference that Ko supports building, returning an error
	// if it is not.
	// TODO(mattmoor): Verify that some base repo: foo.io/bar can be suffixed with this reference and parsed.
	IsSupportedReference(string) error

	// Build turns the given importpath reference into a v1.Image containing the Go binary
	// (or a set of images as a v1.ImageIndex).
	Build(context.Context, string) (Result, error)
}

// Result represents the product of a Build.
// This is generally one of:
// - v1.Image      (or oci.SignedImage), or
// - v1.ImageIndex (or oci.SignedImageIndex)
type Result interface {
	MediaType() (types.MediaType, error)
	Size() (int64, error)
	Digest() (v1.Hash, error)
	RawManifest() ([]byte, error)
}

// Assert that Image and ImageIndex implement Result.
var _ Result = (v1.Image)(nil)
var _ Result = (v1.ImageIndex)(nil)
