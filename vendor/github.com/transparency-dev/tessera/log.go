// Copyright 2024 The Tessera authors. All Rights Reserved.
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

package tessera

import (
	"errors"
)

// ErrPushback is returned by underlying storage implementations when a new entry cannot be accepted
// due to overload in the system. This could be because there are too many entries with indices assigned
// but which have not yet been integrated into the tree, or it could be because the antispam mechanism
// is not able to keep up with recently added entries.
//
// Personalities encountering this error should apply back-pressure to the source of new entries
// in an appropriate manner (e.g. for HTTP services, return a 503 with a Retry-After header).
//
// Personalities should check for this error using `errors.Is(e, ErrPushback)`.
var ErrPushback = errors.New("pushback")

// Driver is the implementation-specific parts of Tessera. No methods are on here as this is not for public use.
type Driver any
