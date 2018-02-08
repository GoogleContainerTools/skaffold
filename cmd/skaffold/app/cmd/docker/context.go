/*
Copyright 2018 Google LLC

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

package docker

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/docker"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var output string

func NewCmdContext(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Outputs a minimal context tarball to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runContext(out, filename, context); err != nil {
				logrus.Fatalf("docker deps: %s", err)
			}
		},
	}
	cmd.Flags().StringVarP(&filename, "filename", "f", "Dockerfile", "Dockerfile path")
	cmd.Flags().StringVarP(&context, "context", "c", ".", "Dockerfile context path")
	cmd.Flags().StringVarP(&output, "output", "o", "context.tar.gz", "Output filename.")
	return cmd
}

func runContext(out io.Writer, filename, context string) error {
	dockerFilePath, err := filepath.Abs(filename)
	logrus.Info(filename)
	logrus.Info(dockerFilePath)
	if err != nil {
		return err
	}
	f, err := os.Open(dockerFilePath)
	if err != nil {
		return errors.Wrap(err, "opening dockerfile")
	}
	deps, err := docker.GetDockerfileDependencies(context, f)
	if err != nil {
		return errors.Wrap(err, "getting dockerfile dependencies")
	}

	// Write everything to memory, then flush to disk at the end.
	// This prevents recursion problems, where the output file can end up
	// in the context itself during creation.
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	defer w.Close()

	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, d := range deps {
		absPath, err := filepath.Abs(d)
		if err != nil {
			return err
		}

		fi, err := os.Lstat(d)
		if err != nil {
			return err
		}
		switch mode := fi.Mode(); {
		case mode.IsRegular():
			tarHeader, err := tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return err
			}

			if absPath == dockerFilePath {
				// The Dockerfile must be placed at the root of the context.
				logrus.Infof("Placing Dockerfile %s at root of context", dockerFilePath)
				tarHeader.Name = "Dockerfile"
			} else {
				tarHeader.Name, err = filepath.Rel(context, d)
				if err != nil {
					return err
				}
			}

			if err := tw.WriteHeader(tarHeader); err != nil {
				return err
			}
			f, err := os.Open(d)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, f); err != nil {
				return errors.Wrapf(err, "writing real file %s", d)
			}
		case mode&os.ModeSymlink != 0:
			target, err := os.Readlink(d)
			if err != nil {
				return err
			}
			tarHeader, err := tar.FileInfoHeader(fi, target)
			if err != nil {
				return err
			}
			if err := tw.WriteHeader(tarHeader); err != nil {
				return err
			}
		default:
			logrus.Warnf("Adding possibly unsupported file %s of type %s.", d, mode)
			// Try to add it anyway?
			tarHeader, err := tar.FileInfoHeader(fi, "")
			if err != nil {
				return err
			}
			if err := tw.WriteHeader(tarHeader); err != nil {
				return err
			}
		}
	}

	// Explicitly close these to flush before writing to disk.
	tw.Close()
	w.Close()
	return ioutil.WriteFile(output, b.Bytes(), 0644)
}
