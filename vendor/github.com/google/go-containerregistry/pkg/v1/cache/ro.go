// Copyright 2021 Google LLC All Rights Reserved.
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

package cache

import v1 "github.com/google/go-containerregistry/pkg/v1"

// ReadOnly returns a read-only implementation of the given Cache.
//
// Put and Delete operations are a no-op.
func ReadOnly(c Cache) Cache { return &ro{Cache: c} }

type ro struct{ Cache }

func (ro) Put(l v1.Layer) (v1.Layer, error) { return l, nil }
func (ro) Delete(v1.Hash) error             { return nil }
