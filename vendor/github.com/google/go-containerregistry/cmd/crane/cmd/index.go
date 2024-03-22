// Copyright 2023 Google LLC All Rights Reserved.
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
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/spf13/cobra"
)

// NewCmdIndex creates a new cobra.Command for the index subcommand.
func NewCmdIndex(options *[]crane.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Modify an image index.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Usage()
		},
	}
	cmd.AddCommand(NewCmdIndexFilter(options), NewCmdIndexAppend(options))
	return cmd
}

// NewCmdIndexFilter creates a new cobra.Command for the index filter subcommand.
func NewCmdIndexFilter(options *[]crane.Option) *cobra.Command {
	var newTag string
	platforms := &platformsValue{}

	cmd := &cobra.Command{
		Use:   "filter",
		Short: "Modifies a remote index by filtering based on platform.",
		Example: `  # Filter out weird platforms from ubuntu, copy result to example.com/ubuntu
  crane index filter ubuntu --platform linux/amd64 --platform linux/arm64 -t example.com/ubuntu

  # Filter out any non-linux platforms, push to example.com/hello-world
  crane index filter hello-world --platform linux -t example.com/hello-world

  # Same as above, but in-place
  crane index filter example.com/hello-world:some-tag --platform linux`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o := crane.GetOptions(*options...)
			baseRef := args[0]

			ref, err := name.ParseReference(baseRef, o.Name...)
			if err != nil {
				return err
			}
			desc, err := remote.Get(ref, o.Remote...)
			if err != nil {
				return fmt.Errorf("pulling %s: %w", baseRef, err)
			}
			if !desc.MediaType.IsIndex() {
				return fmt.Errorf("expected %s to be an index, got %q", baseRef, desc.MediaType)
			}
			base, err := desc.ImageIndex()
			if err != nil {
				return nil
			}

			idx := filterIndex(base, platforms.platforms)

			digest, err := idx.Digest()
			if err != nil {
				return err
			}

			if newTag != "" {
				ref, err = name.ParseReference(newTag, o.Name...)
				if err != nil {
					return fmt.Errorf("parsing reference %s: %w", newTag, err)
				}
			} else {
				if _, ok := ref.(name.Digest); ok {
					ref = ref.Context().Digest(digest.String())
				}
			}

			if err := remote.WriteIndex(ref, idx, o.Remote...); err != nil {
				return fmt.Errorf("pushing image %s: %w", newTag, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), ref.Context().Digest(digest.String()))
			return nil
		},
	}
	cmd.Flags().StringVarP(&newTag, "tag", "t", "", "Tag to apply to resulting image")

	// Consider reusing the persistent flag for this, it's separate so we can have multiple values.
	cmd.Flags().Var(platforms, "platform", "Specifies the platform(s) to keep from base in the form os/arch[/variant][:osversion][,<platform>] (e.g. linux/amd64).")

	return cmd
}

// NewCmdIndexAppend creates a new cobra.Command for the index append subcommand.
func NewCmdIndexAppend(options *[]crane.Option) *cobra.Command {
	var baseRef, newTag string
	var newManifests []string
	var dockerEmptyBase, flatten bool

	cmd := &cobra.Command{
		Use:   "append",
		Short: "Append manifests to a remote index.",
		Long: `This sub-command pushes an index based on an (optional) base index, with appended manifests.

The platform for appended manifests is inferred from the config file or omitted if that is infeasible.`,
		Example: ` # Append a windows hello-world image to ubuntu, push to example.com/hello-world:weird
  crane index append ubuntu -m hello-world@sha256:87b9ca29151260634b95efb84d43b05335dc3ed36cc132e2b920dd1955342d20 -t example.com/hello-world:weird

  # Create an index from scratch for etcd.
  crane index append -m registry.k8s.io/etcd-amd64:3.4.9 -m registry.k8s.io/etcd-arm64:3.4.9 -t example.com/etcd`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				baseRef = args[0]
			}
			o := crane.GetOptions(*options...)

			var (
				base v1.ImageIndex
				err  error
				ref  name.Reference
			)

			if baseRef == "" {
				if newTag == "" {
					return errors.New("at least one of --base or --tag must be specified")
				}

				logs.Warn.Printf("base unspecified, using empty index")
				base = empty.Index
				if dockerEmptyBase {
					base = mutate.IndexMediaType(base, types.DockerManifestList)
				}
			} else {
				ref, err = name.ParseReference(baseRef, o.Name...)
				if err != nil {
					return err
				}
				desc, err := remote.Get(ref, o.Remote...)
				if err != nil {
					return fmt.Errorf("pulling %s: %w", baseRef, err)
				}
				if !desc.MediaType.IsIndex() {
					return fmt.Errorf("expected %s to be an index, got %q", baseRef, desc.MediaType)
				}
				base, err = desc.ImageIndex()
				if err != nil {
					return err
				}
			}

			adds := make([]mutate.IndexAddendum, 0, len(newManifests))

			for _, m := range newManifests {
				ref, err := name.ParseReference(m, o.Name...)
				if err != nil {
					return err
				}
				desc, err := remote.Get(ref, o.Remote...)
				if err != nil {
					return err
				}
				if desc.MediaType.IsImage() {
					img, err := desc.Image()
					if err != nil {
						return err
					}

					cf, err := img.ConfigFile()
					if err != nil {
						return err
					}
					newDesc, err := partial.Descriptor(img)
					if err != nil {
						return err
					}
					newDesc.Platform = cf.Platform()
					adds = append(adds, mutate.IndexAddendum{
						Add:        img,
						Descriptor: *newDesc,
					})
				} else if desc.MediaType.IsIndex() {
					idx, err := desc.ImageIndex()
					if err != nil {
						return err
					}
					if flatten {
						im, err := idx.IndexManifest()
						if err != nil {
							return err
						}
						for _, child := range im.Manifests {
							switch {
							case child.MediaType.IsImage():
								img, err := idx.Image(child.Digest)
								if err != nil {
									return err
								}
								adds = append(adds, mutate.IndexAddendum{
									Add:        img,
									Descriptor: child,
								})
							case child.MediaType.IsIndex():
								idx, err := idx.ImageIndex(child.Digest)
								if err != nil {
									return err
								}
								adds = append(adds, mutate.IndexAddendum{
									Add:        idx,
									Descriptor: child,
								})
							default:
								return fmt.Errorf("unexpected child %q with media type %q", child.Digest, child.MediaType)
							}
						}
					} else {
						adds = append(adds, mutate.IndexAddendum{
							Add: idx,
						})
					}
				} else {
					return fmt.Errorf("saw unexpected MediaType %q for %q", desc.MediaType, m)
				}
			}

			idx := mutate.AppendManifests(base, adds...)
			digest, err := idx.Digest()
			if err != nil {
				return err
			}

			if newTag != "" {
				ref, err = name.ParseReference(newTag, o.Name...)
				if err != nil {
					return fmt.Errorf("parsing reference %s: %w", newTag, err)
				}
			} else {
				if _, ok := ref.(name.Digest); ok {
					ref = ref.Context().Digest(digest.String())
				}
			}

			if err := remote.WriteIndex(ref, idx, o.Remote...); err != nil {
				return fmt.Errorf("pushing image %s: %w", newTag, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), ref.Context().Digest(digest.String()))
			return nil
		},
	}
	cmd.Flags().StringVarP(&newTag, "tag", "t", "", "Tag to apply to resulting image")
	cmd.Flags().StringSliceVarP(&newManifests, "manifest", "m", []string{}, "References to manifests to append to the base index")
	cmd.Flags().BoolVar(&dockerEmptyBase, "docker-empty-base", false, "If true, empty base index will have Docker media types instead of OCI")
	cmd.Flags().BoolVar(&flatten, "flatten", true, "If true, appending an index will append each of its children rather than the index itself")

	return cmd
}

func filterIndex(idx v1.ImageIndex, platforms []v1.Platform) v1.ImageIndex {
	matcher := not(satisfiesPlatforms(platforms))
	return mutate.RemoveManifests(idx, matcher)
}

func satisfiesPlatforms(platforms []v1.Platform) match.Matcher {
	return func(desc v1.Descriptor) bool {
		if desc.Platform == nil {
			return false
		}
		for _, p := range platforms {
			if desc.Platform.Satisfies(p) {
				return true
			}
		}
		return false
	}
}

func not(in match.Matcher) match.Matcher {
	return func(desc v1.Descriptor) bool {
		return !in(desc)
	}
}
