//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mutate

import (
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/sigstore/cosign/pkg/oci"
)

// AppendSignatures produces a new oci.Signatures with the provided signatures
// appended to the provided base signatures.
func AppendSignatures(base oci.Signatures, sigs ...oci.Signature) (oci.Signatures, error) {
	adds := make([]mutate.Addendum, 0, len(sigs))
	for _, sig := range sigs {
		ann, err := sig.Annotations()
		if err != nil {
			return nil, err
		}
		adds = append(adds, mutate.Addendum{
			Layer:       sig,
			Annotations: ann,
		})
	}

	img, err := mutate.Append(base, adds...)
	if err != nil {
		return nil, err
	}

	// Set the Created date to time of execution
	img, err = mutate.CreatedAt(img, v1.Time{Time: time.Now()})
	if err != nil {
		return nil, err
	}

	return &sigAppender{
		Image: img,
		base:  base,
		sigs:  sigs,
	}, nil
}

// ReplaceSignatures produces a new oci.Signatures provided by the base signatures
// replaced with the new oci.Signatures.
func ReplaceSignatures(base oci.Signatures) (oci.Signatures, error) {
	sigs, err := base.Get()
	if err != nil {
		return nil, err
	}
	adds := make([]mutate.Addendum, 0, len(sigs))
	for _, sig := range sigs {
		ann, err := sig.Annotations()
		if err != nil {
			return nil, err
		}
		adds = append(adds, mutate.Addendum{
			Layer:       sig,
			Annotations: ann,
		})
	}
	img, err := mutate.Append(empty.Image, adds...)
	if err != nil {
		return nil, err
	}
	return &sigAppender{
		Image: img,
		base:  base,
		sigs:  sigs,
	}, nil
}

type sigAppender struct {
	v1.Image
	base oci.Signatures
	sigs []oci.Signature
}

var _ oci.Signatures = (*sigAppender)(nil)

// Get implements oci.Signatures
func (sa *sigAppender) Get() ([]oci.Signature, error) {
	sl, err := sa.base.Get()
	if err != nil {
		return nil, err
	}
	return append(sl, sa.sigs...), nil
}
