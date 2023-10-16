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

package publish

import (
	"context"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/ko/pkg/build"
)

// Interface abstracts different methods for publishing images.
type Interface interface {
	// Publish uploads the given build.Result to a registry incorporating the
	// provided string into the image's repository name.  Returns the digest
	// of the published image.
	Publish(context.Context, build.Result, string) (name.Reference, error)

	// Close exists for the tarball implementation so we can
	// do the whole thing in one write.
	Close() error
}
