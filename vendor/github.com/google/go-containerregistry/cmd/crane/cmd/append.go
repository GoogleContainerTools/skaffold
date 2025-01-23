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
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

// NewCmdAppend creates a new cobra.Command for the append subcommand.
func NewCmdAppend(options *[]crane.Option) *cobra.Command {
	var baseRef, newTag, outFile string
	var newLayers []string
	var annotate, ociEmptyBase bool

	appendCmd := &cobra.Command{
		Use:   "append",
		Short: "Append contents of a tarball to a remote image",
		Long: `This sub-command pushes an image based on an (optional)
base image, with appended layers containing the contents of the
provided tarballs.

If the base image is a Windows base image (i.e., its config.OS is "windows"),
the contents of the tarballs will be modified to be suitable for a Windows
container image.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var base v1.Image
			var err error

			if baseRef == "" {
				logs.Warn.Printf("base unspecified, using empty image")
				base = empty.Image
				if ociEmptyBase {
					base = mutate.MediaType(base, types.OCIManifestSchema1)
					base = mutate.ConfigMediaType(base, types.OCIConfigJSON)
				}
			} else {
				base, err = crane.Pull(baseRef, *options...)
				if err != nil {
					return fmt.Errorf("pulling %s: %w", baseRef, err)
				}
			}

			img, err := crane.Append(base, newLayers...)
			if err != nil {
				return fmt.Errorf("appending %v: %w", newLayers, err)
			}

			if baseRef != "" && annotate {
				ref, err := name.ParseReference(baseRef)
				if err != nil {
					return fmt.Errorf("parsing ref %q: %w", baseRef, err)
				}

				baseDigest, err := base.Digest()
				if err != nil {
					return err
				}
				anns := map[string]string{
					specsv1.AnnotationBaseImageDigest: baseDigest.String(),
				}
				if _, ok := ref.(name.Tag); ok {
					anns[specsv1.AnnotationBaseImageName] = ref.Name()
				}
				img = mutate.Annotations(img, anns).(v1.Image)
			}

			if outFile != "" {
				if err := crane.Save(img, newTag, outFile); err != nil {
					return fmt.Errorf("writing output %q: %w", outFile, err)
				}
			} else {
				if err := crane.Push(img, newTag, *options...); err != nil {
					return fmt.Errorf("pushing image %s: %w", newTag, err)
				}
				ref, err := name.ParseReference(newTag)
				if err != nil {
					return fmt.Errorf("parsing reference %s: %w", newTag, err)
				}
				d, err := img.Digest()
				if err != nil {
					return fmt.Errorf("digest: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), ref.Context().Digest(d.String()))
			}
			return nil
		},
	}
	appendCmd.Flags().StringVarP(&baseRef, "base", "b", "", "Name of base image to append to")
	appendCmd.Flags().StringVarP(&newTag, "new_tag", "t", "", "Tag to apply to resulting image")
	appendCmd.Flags().StringSliceVarP(&newLayers, "new_layer", "f", []string{}, "Path to tarball to append to image")
	appendCmd.Flags().StringVarP(&outFile, "output", "o", "", "Path to new tarball of resulting image")
	appendCmd.Flags().BoolVar(&annotate, "set-base-image-annotations", false, "If true, annotate the resulting image as being based on the base image")
	appendCmd.Flags().BoolVar(&ociEmptyBase, "oci-empty-base", false, "If true, empty base image will have OCI media types instead of Docker")

	appendCmd.MarkFlagsMutuallyExclusive("oci-empty-base", "base")
	appendCmd.MarkFlagRequired("new_tag")
	appendCmd.MarkFlagRequired("new_layer")
	return appendCmd
}
