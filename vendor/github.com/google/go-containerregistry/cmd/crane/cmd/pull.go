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
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

// NewCmdPull creates a new cobra.Command for the pull subcommand.
func NewCmdPull(options *[]crane.Option) *cobra.Command {
	var (
		cachePath, format string
		annotateRef       bool
	)

	cmd := &cobra.Command{
		Use:   "pull IMAGE TARBALL",
		Short: "Pull remote images by reference and store their contents locally",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			imageMap := map[string]v1.Image{}
			indexMap := map[string]v1.ImageIndex{}
			srcList, path := args[:len(args)-1], args[len(args)-1]
			o := crane.GetOptions(*options...)
			for _, src := range srcList {
				ref, err := name.ParseReference(src, o.Name...)
				if err != nil {
					return fmt.Errorf("parsing reference %q: %w", src, err)
				}

				rmt, err := remote.Get(ref, o.Remote...)
				if err != nil {
					return err
				}

				// If we're writing an index to a layout and --platform hasn't been set,
				// pull the entire index, not just a child image.
				if format == "oci" && rmt.MediaType.IsIndex() && o.Platform == nil {
					idx, err := rmt.ImageIndex()
					if err != nil {
						return err
					}
					indexMap[src] = idx
					continue
				}

				img, err := rmt.Image()
				if err != nil {
					return err
				}
				if cachePath != "" {
					img = cache.Image(img, cache.NewFilesystemCache(cachePath))
				}
				imageMap[src] = img
			}

			switch format {
			case "tarball":
				if err := crane.MultiSave(imageMap, path); err != nil {
					return fmt.Errorf("saving tarball %s: %w", path, err)
				}
			case "legacy":
				if err := crane.MultiSaveLegacy(imageMap, path); err != nil {
					return fmt.Errorf("saving legacy tarball %s: %w", path, err)
				}
			case "oci":
				// Don't use crane.MultiSaveOCI so we can control annotations.
				p, err := layout.FromPath(path)
				if err != nil {
					p, err = layout.Write(path, empty.Index)
					if err != nil {
						return err
					}
				}
				for ref, img := range imageMap {
					opts := []layout.Option{}
					if annotateRef {
						parsed, err := name.ParseReference(ref, o.Name...)
						if err != nil {
							return err
						}
						opts = append(opts, layout.WithAnnotations(map[string]string{
							"org.opencontainers.image.ref.name": parsed.Name(),
						}))
					}
					if err = p.AppendImage(img, opts...); err != nil {
						return err
					}
				}

				for ref, idx := range indexMap {
					opts := []layout.Option{}
					if annotateRef {
						parsed, err := name.ParseReference(ref, o.Name...)
						if err != nil {
							return err
						}
						opts = append(opts, layout.WithAnnotations(map[string]string{
							"org.opencontainers.image.ref.name": parsed.Name(),
						}))
					}
					if err := p.AppendIndex(idx, opts...); err != nil {
						return err
					}
				}
			default:
				return fmt.Errorf("unexpected --format: %q (valid values are: tarball, legacy, and oci)", format)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&cachePath, "cache_path", "c", "", "Path to cache image layers")
	cmd.Flags().StringVar(&format, "format", "tarball", fmt.Sprintf("Format in which to save images (%q, %q, or %q)", "tarball", "legacy", "oci"))
	cmd.Flags().BoolVar(&annotateRef, "annotate-ref", false, "Preserves image reference used to pull as an annotation when used with --format=oci")

	return cmd
}
