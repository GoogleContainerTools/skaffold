// Copyright 2019 ko Build Authors All Rights Reserved.
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

	"golang.org/x/sync/semaphore"
)

// Limiter composes with another Interface to limit the number of concurrent builds.
type Limiter struct {
	Builder   Interface
	semaphore *semaphore.Weighted
}

// Limiter implements Interface
var _ Interface = (*Recorder)(nil)

// QualifyImport implements Interface
func (l *Limiter) QualifyImport(ip string) (string, error) {
	return l.Builder.QualifyImport(ip)
}

// IsSupportedReference implements Interface
func (l *Limiter) IsSupportedReference(ip string) error {
	return l.Builder.IsSupportedReference(ip)
}

// Build implements Interface
func (l *Limiter) Build(ctx context.Context, ip string) (Result, error) {
	if err := l.semaphore.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer l.semaphore.Release(1)

	return l.Builder.Build(ctx, ip)
}

// NewLimiter returns a new builder that only allows n concurrent builds of b.
//
// Deprecated: Obsoleted by WithJobs option.
func NewLimiter(b Interface, n int) *Limiter {
	return &Limiter{
		Builder:   b,
		semaphore: semaphore.NewWeighted(int64(n)),
	}
}
