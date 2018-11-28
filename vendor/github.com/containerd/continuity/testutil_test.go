/*
   Copyright The containerd Authors.

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

package continuity

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func tree(w io.Writer, dir string) error {
	fmt.Fprintf(w, "%s\n", dir)
	return _tree(w, dir, "")
}

func _tree(w io.Writer, dir string, indent string) error {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for i, f := range files {
		fPath := filepath.Join(dir, f.Name())
		fIndent := indent
		if i < len(files)-1 {
			fIndent += "|-- "
		} else {
			fIndent += "`-- "
		}
		target := ""
		if f.Mode()&os.ModeSymlink == os.ModeSymlink {
			realPath, err := os.Readlink(fPath)
			if err != nil {
				target += fmt.Sprintf(" -> unknown (error: %v)", err)
			} else {
				target += " -> " + realPath
			}
		}
		fmt.Fprintf(w, "%s%s%s\n",
			fIndent, f.Name(), target)
		if f.IsDir() {
			dIndent := indent
			if i < len(files)-1 {
				dIndent += "|   "
			} else {
				dIndent += "    "
			}
			if err := _tree(w, fPath, dIndent); err != nil {
				return err
			}
		}
	}
	return nil
}
