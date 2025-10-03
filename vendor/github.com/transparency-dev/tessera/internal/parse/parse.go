// Copyright 2024 The Tessera authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package parse contains internal methods for parsing data structures quickly,
// if unsafely. This is a bit of a utility package which is an anti-pattern, but
// this code is critical enough that it should be reused, tested, and benchmarked
// rather than copied around willy nilly.
// If a better home becomes available, feel free to move the contents elsewhere.
package parse

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strconv"
)

// CheckpointUnsafe parses a checkpoint without performing any signature verification.
// This is intended to be as fast as possible, but sacrifices safety because it skips verifying
// the note signature.
//
// Parsing a checkpoint like this is only acceptable in the same binary as the
// log implementation that generated it and thus we can safely assume it's a well formed and
// validly signed checkpoint. Anyone copying similar logic into client code will get hurt.
func CheckpointUnsafe(rawCp []byte) (string, uint64, []byte, error) {
	parts := bytes.SplitN(rawCp, []byte{'\n'}, 4)
	if want, got := 4, len(parts); want != got {
		return "", 0, nil, fmt.Errorf("invalid checkpoint: %q", rawCp)
	}
	origin := string(parts[0])
	sizeStr := string(parts[1])
	hashStr := string(parts[2])
	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return "", 0, nil, fmt.Errorf("failed to turn checkpoint size of %q into uint64: %v", sizeStr, err)
	}
	hash, err := base64.StdEncoding.DecodeString(hashStr)
	if err != nil {
		return "", 0, nil, fmt.Errorf("failed to decode hash: %v", err)
	}
	return origin, size, hash, nil
}
