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
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
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
	cmd.AddCommand(NewCmdIndexFilter(options), NewCmdIndexAppend(options), NewCmdIndexList(options))
	return cmd
}

// NewCmdIndexList creates a new cobra.Command for the index list subcommand.
func NewCmdIndexList(options *[]crane.Option) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the manifests in an index.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseRef := args[0]
			o := crane.GetOptions(*options...)

			isLocal := isLocalReference(baseRef)

			var idx v1.ImageIndex
			if isLocal {
				p, err := layout.FromPath(baseRef)
				if err != nil {
					return err
				}
				idx, err = p.ImageIndex()
				if err != nil {
					return err
				}
			} else {
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
				idx, err = desc.ImageIndex()
				if err != nil {
					return err
				}
			}

			m, err := idx.IndexManifest()
			if err != nil {
				return err
			}

			// Print header
			fmt.Fprintf(cmd.OutOrStdout(), "%-70s %-20s %s\n", "Digest", "MediaType", "Platform")
			for _, manifest := range m.Manifests {
				platform := "-"
				if manifest.Platform != nil {
					platform = manifest.Platform.String()
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%-70s %-20s %s\n", manifest.Digest, manifest.MediaType, platform)
			}
			return nil
		},
	}
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
				// If we can't get the image index, we just return nil to silently stop.
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				baseRef = args[0]
				newManifests = append(newManifests, args[1:]...)
			}
			o := crane.GetOptions(*options...)

			adds, err := collectAddendums(newManifests, o, flatten)
			if err != nil {
				return err
			}

			if isLocalReference(baseRef) {
				return appendLocal(baseRef, adds)
			}
			return appendRemote(cmd, baseRef, newTag, adds, o, dockerEmptyBase)
		},
	}
	cmd.Flags().StringVarP(&newTag, "tag", "t", "", "Tag to apply to resulting image")
	cmd.Flags().StringSliceVarP(&newManifests, "manifest", "m", []string{}, "References to manifests to append to the base index")
	cmd.Flags().BoolVar(&dockerEmptyBase, "docker-empty-base", false, "If true, empty base index will have Docker media types instead of OCI")
	cmd.Flags().BoolVar(&flatten, "flatten", true, "If true, appending an index will append each of its children rather than the index itself")

	return cmd
}

func appendLocal(baseRef string, adds []mutate.IndexAddendum) error {
	var p layout.Path
	if _, err := os.Stat(baseRef); err == nil {
		// If the path exists, try to open it as an OCI layout.
		var err error
		p, err = layout.FromPath(baseRef)
		if err != nil {
			return err
		}
	} else {
		// If the path does not exist, initialize a new OCI layout.
		var err error
		p, err = layout.Write(baseRef, empty.Index)
		if err != nil {
			return err
		}
	}

	for _, add := range adds {
		opts := []layout.Option{}
		if add.Platform != nil {
			opts = append(opts, layout.WithPlatform(*add.Platform))
		}

		if add.MediaType.IsImage() {
			img, ok := add.Add.(v1.Image)
			if !ok {
				return fmt.Errorf("internal error: add.Add is not v1.Image: %T", add.Add)
			}
			if err := p.AppendImage(img, opts...); err != nil {
				return err
			}
		} else if add.MediaType.IsIndex() {
			idx, ok := add.Add.(v1.ImageIndex)
			if !ok {
				return fmt.Errorf("internal error: add.Add is not v1.ImageIndex: %T", add.Add)
			}
			if err := p.AppendIndex(idx, opts...); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unexpected media type for local append: %s", add.MediaType)
		}
	}
	return nil
}

func appendRemote(cmd *cobra.Command, baseRef, newTag string, adds []mutate.IndexAddendum, o crane.Options, dockerEmptyBase bool) error {
	var base v1.ImageIndex
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
		base, err = desc.ImageIndex()
		if err != nil {
			return err
		}
	}

	idx := mutate.AppendManifests(base, adds...)
	digest, err := idx.Digest()
	if err != nil {
		return err
	}

	var targetRef name.Reference
	if newTag != "" {
		targetRef, err = name.ParseReference(newTag, o.Name...)
		if err != nil {
			return fmt.Errorf("parsing reference %s: %w", newTag, err)
		}
	} else {
		if baseRef != "" {
			var err error
			targetRef, err = name.ParseReference(baseRef, o.Name...)
			if err != nil {
				return err
			}
			if _, ok := targetRef.(name.Digest); ok {
				targetRef = targetRef.Context().Digest(digest.String())
			}
		}
	}

	if targetRef == nil {
		return errors.New("no target reference determined")
	}

	if err := remote.WriteIndex(targetRef, idx, o.Remote...); err != nil {
		return fmt.Errorf("pushing image %s: %w", targetRef, err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), targetRef.Context().Digest(digest.String()))
	return nil
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

// resolveManifest resolves a reference to either a local OCI layout or a remote manifest.
// It returns the manifest (Image or ImageIndex) and its descriptor.
func resolveManifest(ref string, o crane.Options) (partial.WithRawManifest, v1.Descriptor, error) {
	// Try loading as local image first. If this fails, we fall back to remote.
	img, err := loadImage(ref, true)
	if err == nil {
		desc, err := partial.Descriptor(img.(partial.Describable))
		if err != nil {
			return nil, v1.Descriptor{}, err
		}
		// If it's an image, try to get platform from config.
		// Platform info is not in the manifest, but is required for the index descriptor.
		if desc.MediaType.IsImage() {
			if i, ok := img.(v1.Image); ok {
				cf, err := i.ConfigFile()
				if err != nil {
					return nil, v1.Descriptor{}, err
				}
				desc.Platform = cf.Platform()
			}
		}
		return img, *desc, nil
	}

	// Fallback to remote
	r, err := name.ParseReference(ref, o.Name...)
	if err != nil {
		return nil, v1.Descriptor{}, err
	}
	d, err := remote.Get(r, o.Remote...)
	if err != nil {
		return nil, v1.Descriptor{}, err
	}

	var remoteImg partial.WithRawManifest
	if d.MediaType.IsImage() {
		i, err := d.Image()
		if err != nil {
			return nil, v1.Descriptor{}, err
		}
		// Populate platform info from the config blob for the index descriptor.
		cf, err := i.ConfigFile()
		if err != nil {
			return nil, v1.Descriptor{}, err
		}
		d.Platform = cf.Platform()
		remoteImg = i
	} else if d.MediaType.IsIndex() {
		idx, err := d.ImageIndex()
		if err != nil {
			return nil, v1.Descriptor{}, err
		}
		remoteImg = idx
	} else {
		return nil, v1.Descriptor{}, fmt.Errorf("unknown media type: %s", d.MediaType)
	}
	return remoteImg, d.Descriptor, nil
}

// collectAddendums resolves a list of manifest references (local or remote) and
// returns a slice of IndexAddendums that can be appended to an index.
// If flatten is true, any referenced indices will have their individual child
// manifests added to the result rather than the index itself.
func collectAddendums(manifests []string, o crane.Options, flatten bool) ([]mutate.IndexAddendum, error) {
	var adds []mutate.IndexAddendum
	for _, m := range manifests {
		img, desc, err := resolveManifest(m, o)
		if err != nil {
			return nil, err
		}

		if desc.MediaType.IsImage() {
			i, ok := img.(v1.Image)
			if !ok {
				return nil, fmt.Errorf("expected v1.Image, got %T", img)
			}
			adds = append(adds, mutate.IndexAddendum{
				Add:        i,
				Descriptor: desc,
			})
		} else if desc.MediaType.IsIndex() {
			idx, ok := img.(v1.ImageIndex)
			if !ok {
				return nil, fmt.Errorf("expected v1.ImageIndex, got %T", img)
			}
			if flatten {
				im, err := idx.IndexManifest()
				if err != nil {
					return nil, err
				}
				for _, child := range im.Manifests {
					if child.MediaType.IsImage() {
						childImg, err := idx.Image(child.Digest)
						if err != nil {
							return nil, err
						}
						adds = append(adds, mutate.IndexAddendum{
							Add:        childImg,
							Descriptor: child,
						})
					} else if child.MediaType.IsIndex() {
						childIdx, err := idx.ImageIndex(child.Digest)
						if err != nil {
							return nil, err
						}
						adds = append(adds, mutate.IndexAddendum{
							Add:        childIdx,
							Descriptor: child,
						})
					} else {
						return nil, fmt.Errorf("unexpected child media type: %s", child.MediaType)
					}
				}
			} else {
				adds = append(adds, mutate.IndexAddendum{
					Add:        idx,
					Descriptor: desc,
				})
			}
		} else {
			return nil, fmt.Errorf("unexpected media type: %s", desc.MediaType)
		}
	}
	return adds, nil
}

// isLocalReference checks if the reference is a local file path.
// It returns true if the path exists on the filesystem.
// If the path does NOT exist, it still returns true if it looks like a path
// (starts with . or / or \ on Windows). This is necessary because
// 'crane index append' can create new local OCI layout directories that
// do not yet exist, and we need to distinguish these from remote references.
func isLocalReference(ref string) bool {
	if _, err := os.Stat(ref); err == nil {
		return true
	}
	return strings.HasPrefix(ref, ".") || strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, string(os.PathSeparator))
}
