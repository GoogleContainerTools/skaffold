// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package contains a program that generates code to register
// a directory and its contents as zip data for statik file system.
package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	nameSourceFile = "statik.go"
)

var namePackage string

var (
	flagSrc        = flag.String("src", path.Join(".", "public"), "The path of the source directory.")
	flagDest       = flag.String("dest", ".", "The destination path of the generated package.")
	flagNoMtime    = flag.Bool("m", false, "Ignore modification times on files.")
	flagNoCompress = flag.Bool("Z", false, "Do not use compression to shrink the files.")
	flagForce      = flag.Bool("f", false, "Overwrite destination file if it already exists.")
	flagTags       = flag.String("tags", "", "Write build constraint tags")
	flagPkg        = flag.String("p", "statik", "Name of the generated package")
	flagPkgCmt     = flag.String("c", "Package statik contains static assets.", "The package comment. An empty value disables this comment.\n")
)

// mtimeDate holds the arbitrary mtime that we assign to files when
// flagNoMtime is set.
var mtimeDate = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

func main() {
	flag.Parse()

	namePackage = *flagPkg

	file, err := generateSource(*flagSrc)
	if err != nil {
		exitWithError(err)
	}

	destDir := path.Join(*flagDest, namePackage)
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		exitWithError(err)
	}

	err = rename(file.Name(), path.Join(destDir, nameSourceFile))
	if err != nil {
		exitWithError(err)
	}
}

// rename tries to os.Rename, but fall backs to copying from src
// to dest and unlink the source if os.Rename fails.
func rename(src, dest string) error {
	// Try to rename generated source.
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	// If the rename failed (might do so due to temporary file residing on a
	// different device), try to copy byte by byte.
	rc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		rc.Close()
		os.Remove(src) // ignore the error, source is in tmp.
	}()

	if _, err = os.Stat(dest); !os.IsNotExist(err) {
		if *flagForce {
			if err = os.Remove(dest); err != nil {
				return fmt.Errorf("file %q could not be deleted", dest)
			}
		} else {
			return fmt.Errorf("file %q already exists; use -f to overwrite", dest)
		}
	}

	wc, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer wc.Close()

	if _, err = io.Copy(wc, rc); err != nil {
		// Delete remains of failed copy attempt.
		os.Remove(dest)
	}
	return err
}

// Walks on the source path and generates source code
// that contains source directory's contents as zip contents.
// Generates source registers generated zip contents data to
// be read by the statik/fs HTTP file system.
func generateSource(srcPath string) (file *os.File, err error) {
	var (
		buffer    bytes.Buffer
		zipWriter io.Writer
	)

	zipWriter = &buffer
	f, err := ioutil.TempFile("", namePackage)
	if err != nil {
		return
	}

	zipWriter = io.MultiWriter(zipWriter, f)
	defer f.Close()

	w := zip.NewWriter(zipWriter)
	if err = filepath.Walk(srcPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Ignore directories and hidden files.
		// No entry is needed for directories in a zip file.
		// Each file is represented with a path, no directory
		// entities are required to build the hierarchy.
		if fi.IsDir() || strings.HasPrefix(fi.Name(), ".") {
			return nil
		}
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		fHeader, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}
		if *flagNoMtime {
			// Always use the same modification time so that
			// the output is deterministic with respect to the file contents.
			// Do NOT use fHeader.Modified as it only works on go >= 1.10
			fHeader.SetModTime(mtimeDate)
		}
		fHeader.Name = filepath.ToSlash(relPath)
		if !*flagNoCompress {
			fHeader.Method = zip.Deflate
		}
		f, err := w.CreateHeader(fHeader)
		if err != nil {
			return err
		}
		_, err = f.Write(b)
		return err
	}); err != nil {
		return
	}
	if err = w.Close(); err != nil {
		return
	}

	var tags string
	if *flagTags != "" {
		tags = "\n// +build " + *flagTags + "\n"
	}

	var comment string
	if *flagPkgCmt != "" {
		comment = "\n" + commentLines(*flagPkgCmt)
	}

	// then embed it as a quoted string
	var qb bytes.Buffer
	fmt.Fprintf(&qb, `// Code generated by statik. DO NOT EDIT.
%s%s
package %s

import (
	"github.com/rakyll/statik/fs"
)

func init() {
	data := "`, tags, comment, namePackage)
	FprintZipData(&qb, buffer.Bytes())
	fmt.Fprint(&qb, `"
	fs.Register(data)
}
`)

	if err = ioutil.WriteFile(f.Name(), qb.Bytes(), 0644); err != nil {
		return
	}
	return f, nil
}

// FprintZipData converts zip binary contents to a string literal.
func FprintZipData(dest *bytes.Buffer, zipData []byte) {
	for _, b := range zipData {
		if b == '\n' {
			dest.WriteString(`\n`)
			continue
		}
		if b == '\\' {
			dest.WriteString(`\\`)
			continue
		}
		if b == '"' {
			dest.WriteString(`\"`)
			continue
		}
		if (b >= 32 && b <= 126) || b == '\t' {
			dest.WriteByte(b)
			continue
		}
		fmt.Fprintf(dest, "\\x%02x", b)
	}
}

// comment lines prefixes each line in lines with "// ".
func commentLines(lines string) string {
	lines = "// " + strings.Replace(lines, "\n", "\n// ", -1)
	return lines
}

// Prints out the error message and exists with a non-success signal.
func exitWithError(err error) {
	fmt.Println(err)
	os.Exit(1)
}
