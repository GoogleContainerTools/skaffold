// Copyright 2024 ko Build Authors All Rights Reserved.
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

// MIT License
//
// Copyright (c) 2016-2022 Carlos Alexandro Becker
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package git

import (
	"errors"
	"fmt"
)

var (
	// ErrNoTag happens if the underlying git repository doesn't contain any tags
	// but no snapshot-release was requested.
	ErrNoTag = errors.New("git doesn't contain any tags. Tag info will not be available")

	// ErrNotRepository happens if you try to run ko against a folder
	// which is not a git repository.
	ErrNotRepository = errors.New("current folder is not a git repository. Git info will not be available")

	// ErrNoGit happens when git is not present in PATH.
	ErrNoGit = errors.New("git not present in PATH. Git info will not be available")
)

// ErrDirty happens when the repo has uncommitted/unstashed changes.
type ErrDirty struct {
	status string
}

func (e ErrDirty) Error() string {
	return fmt.Sprintf("git is in a dirty state\nPlease check in your pipeline what can be changing the following files:\n%v\n", e.status)
}
