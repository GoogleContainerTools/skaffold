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
	"errors"
	"fmt"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

// NewCmdDigest creates a new cobra.Command for the digest subcommand.
func NewCmdDigest(options *[]crane.Option) *cobra.Command {
	var tarball string
	var fullRef bool
	cmd := &cobra.Command{
		Use:   "digest IMAGE",
		Short: "Get the digest of an image",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tarball == "" && len(args) == 0 {
				if err := cmd.Help(); err != nil {
					return err
				}
				return errors.New("image reference required without --tarball")
			}
			if fullRef && tarball != "" {
				return errors.New("cannot specify --full-ref with --tarball")
			}

			digest, err := getDigest(tarball, args, options)
			if err != nil {
				return err
			}
			if fullRef {
				ref, err := name.ParseReference(args[0])
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), ref.Context().Digest(digest))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), digest)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&tarball, "tarball", "", "(Optional) path to tarball containing the image")
	cmd.Flags().BoolVar(&fullRef, "full-ref", false, "(Optional) if true, print the full image reference by digest")

	return cmd
}

func getDigest(tarball string, args []string, options *[]crane.Option) (string, error) {
	if tarball != "" {
		return getTarballDigest(tarball, args, options)
	}

	return crane.Digest(args[0], *options...)
}

func getTarballDigest(tarball string, args []string, options *[]crane.Option) (string, error) {
	tag := ""
	if len(args) > 0 {
		tag = args[0]
	}

	img, err := crane.LoadTag(tarball, tag, *options...)
	if err != nil {
		return "", fmt.Errorf("loading image from %q: %w", tarball, err)
	}
	digest, err := img.Digest()
	if err != nil {
		return "", fmt.Errorf("computing digest: %w", err)
	}
	return digest.String(), nil
}
