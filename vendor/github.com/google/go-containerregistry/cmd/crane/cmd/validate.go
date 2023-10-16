// Copyright 2018 Google LLC All Rights Reserved.
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

package cmd

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/spf13/cobra"
)

// NewCmdValidate creates a new cobra.Command for the validate subcommand.
func NewCmdValidate(options *[]crane.Option) *cobra.Command {
	var (
		tarballPath, remoteRef string
		fast                   bool
	)

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate that an image is well-formed",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			for flag, maker := range map[string]func(string, ...crane.Option) (v1.Image, error){
				tarballPath: makeTarball,
				remoteRef:   crane.Pull,
			} {
				if flag == "" {
					continue
				}
				img, err := maker(flag, *options...)
				if err != nil {
					return fmt.Errorf("failed to read image %s: %w", flag, err)
				}

				opt := []validate.Option{}
				if fast {
					opt = append(opt, validate.Fast)
				}
				if err := validate.Image(img, opt...); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "FAIL: %s: %v\n", flag, err)
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "PASS: %s\n", flag)
			}
			return nil
		},
	}
	validateCmd.Flags().StringVar(&tarballPath, "tarball", "", "Path to tarball to validate")
	validateCmd.Flags().StringVar(&remoteRef, "remote", "", "Name of remote image to validate")
	validateCmd.Flags().BoolVar(&fast, "fast", false, "Skip downloading/digesting layers")

	return validateCmd
}

func makeTarball(path string, _ ...crane.Option) (v1.Image, error) {
	return tarball.ImageFromPath(path, nil)
}
