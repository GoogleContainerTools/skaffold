// Copyright 2021 Google LLC All Rights Reserved.
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
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/spf13/cobra"
)

// NewCmdMutate creates a new cobra.Command for the mutate subcommand.
func NewCmdMutate(options *[]crane.Option) *cobra.Command {
	var labels map[string]string
	var annotations map[string]string
	var envVars keyToValue
	var entrypoint, cmd []string
	var newLayers []string
	var outFile string
	var newRef string
	var newRepo string
	var user string
	var workdir string
	var ports []string

	mutateCmd := &cobra.Command{
		Use:   "mutate",
		Short: "Modify image labels and annotations. The container must be pushed to a registry, and the manifest is updated there.",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			// Pull image and get config.
			ref := args[0]

			if len(annotations) != 0 {
				desc, err := crane.Head(ref, *options...)
				if err != nil {
					return err
				}
				if desc.MediaType.IsIndex() {
					return errors.New("mutating annotations on an index is not yet supported")
				}
			}

			if newRepo != "" && newRef != "" {
				return errors.New("repository can't be set when a tag is specified")
			}

			img, err := crane.Pull(ref, *options...)
			if err != nil {
				return fmt.Errorf("pulling %s: %w", ref, err)
			}
			if len(newLayers) != 0 {
				img, err = crane.Append(img, newLayers...)
				if err != nil {
					return fmt.Errorf("appending %v: %w", newLayers, err)
				}
			}
			cfg, err := img.ConfigFile()
			if err != nil {
				return err
			}
			cfg = cfg.DeepCopy()

			// Set labels.
			if cfg.Config.Labels == nil {
				cfg.Config.Labels = map[string]string{}
			}

			if err := validateKeyVals(labels); err != nil {
				return err
			}

			for k, v := range labels {
				cfg.Config.Labels[k] = v
			}

			if err := validateKeyVals(annotations); err != nil {
				return err
			}

			// set envvars if specified
			if err := setEnvVars(cfg, envVars); err != nil {
				return err
			}

			// Set entrypoint.
			if len(entrypoint) > 0 {
				cfg.Config.Entrypoint = entrypoint
				cfg.Config.Cmd = nil // This matches Docker's behavior.
			}

			// Set cmd.
			if len(cmd) > 0 {
				cfg.Config.Cmd = cmd
			}

			// Set user.
			if len(user) > 0 {
				cfg.Config.User = user
			}

			// Set workdir.
			if len(workdir) > 0 {
				cfg.Config.WorkingDir = workdir
			}

			// Set ports
			if len(ports) > 0 {
				portMap := make(map[string]struct{})
				for _, port := range ports {
					portMap[port] = struct{}{}
				}
				cfg.Config.ExposedPorts = portMap
			}

			// Mutate and write image.
			img, err = mutate.Config(img, cfg.Config)
			if err != nil {
				return fmt.Errorf("mutating config: %w", err)
			}

			img = mutate.Annotations(img, annotations).(v1.Image)

			// If the new ref isn't provided, write over the original image.
			// If that ref was provided by digest (e.g., output from
			// another crane command), then strip that and push the
			// mutated image by digest instead.
			if newRepo != "" {
				newRef = newRepo
			} else if newRef == "" {
				newRef = ref
			}
			digest, err := img.Digest()
			if err != nil {
				return fmt.Errorf("digesting new image: %w", err)
			}
			if outFile != "" {
				if err := crane.Save(img, newRef, outFile); err != nil {
					return fmt.Errorf("writing output %q: %w", outFile, err)
				}
			} else {
				r, err := name.ParseReference(newRef)
				if err != nil {
					return fmt.Errorf("parsing %s: %w", newRef, err)
				}
				if _, ok := r.(name.Digest); ok || newRepo != "" {
					newRef = r.Context().Digest(digest.String()).String()
				}
				if err := crane.Push(img, newRef, *options...); err != nil {
					return fmt.Errorf("pushing %s: %w", newRef, err)
				}
				fmt.Fprintln(c.OutOrStdout(), r.Context().Digest(digest.String()))
			}
			return nil
		},
	}
	mutateCmd.Flags().StringToStringVarP(&annotations, "annotation", "a", nil, "New annotations to add")
	mutateCmd.Flags().StringToStringVarP(&labels, "label", "l", nil, "New labels to add")
	mutateCmd.Flags().VarP(&envVars, "env", "e", "New envvar to add")
	mutateCmd.Flags().StringSliceVar(&entrypoint, "entrypoint", nil, "New entrypoint to set")
	mutateCmd.Flags().StringSliceVar(&cmd, "cmd", nil, "New cmd to set")
	mutateCmd.Flags().StringVar(&newRepo, "repo", "", "Repository to push the mutated image to. If provided, push by digest to this repository.")
	mutateCmd.Flags().StringVarP(&newRef, "tag", "t", "", "New tag reference to apply to mutated image. If not provided, push by digest to the original image repository.")
	mutateCmd.Flags().StringVarP(&outFile, "output", "o", "", "Path to new tarball of resulting image")
	mutateCmd.Flags().StringSliceVar(&newLayers, "append", []string{}, "Path to tarball to append to image")
	mutateCmd.Flags().StringVarP(&user, "user", "u", "", "New user to set")
	mutateCmd.Flags().StringVarP(&workdir, "workdir", "w", "", "New working dir to set")
	mutateCmd.Flags().StringSliceVar(&ports, "exposed-ports", nil, "New ports to expose")
	return mutateCmd
}

// validateKeyVals ensures no values are empty, returns error if they are
func validateKeyVals(kvPairs map[string]string) error {
	for label, value := range kvPairs {
		if value == "" {
			return fmt.Errorf("parsing label %q, value is empty", label)
		}
	}
	return nil
}

// setEnvVars override envvars in a config
func setEnvVars(cfg *v1.ConfigFile, envVars keyToValue) error {
	eMap := envVars.Map()
	newEnv := make([]string, 0, len(cfg.Config.Env))
	isWindows := cfg.OS == "windows"

	// Keep the old values.
	for _, old := range cfg.Config.Env {
		oldKey, _, ok := strings.Cut(old, "=")
		if !ok {
			return fmt.Errorf("invalid key value pair in config: %s", old)
		}

		if v, ok := eMap[oldKey]; ok {
			// Override in place to keep ordering of original env.
			newEnv = append(newEnv, oldKey+"="+v)

			// Remove this from eMap so we don't add it twice.
			delete(eMap, oldKey)
		} else {
			newEnv = append(newEnv, old)
		}
	}

	// Append the new values.
	for _, e := range envVars.values {
		k, v := e.key, e.value

		if _, ok := eMap[k]; !ok {
			// If we come across a value not in eMap, it means we replaced the
			// old env in-place and deleted it from eMap, so we can skip adding.
			continue
		}

		if isWindows {
			k = strings.ToUpper(k)
		}

		newEnv = append(newEnv, fmt.Sprintf("%s=%s", k, v))
	}

	cfg.Config.Env = newEnv
	return nil
}

type env struct {
	key   string
	value string
}

type keyToValue struct {
	values  []env
	changed bool
	mapped  map[string]string
}

func (o *keyToValue) Set(val string) error {
	before, after, ok := strings.Cut(val, "=")
	if !ok {
		return fmt.Errorf("%s must be formatted as key=value", val)
	}

	if !o.changed {
		o.values = []env{}
		o.mapped = map[string]string{}
	}

	o.values = append(o.values, env{before, after})
	o.mapped[before] = after
	o.changed = true

	return nil
}

func (o *keyToValue) Type() string {
	return "keyToValue"
}

func (o *keyToValue) String() string {
	ss := make([]string, 0, len(o.values))
	for _, e := range o.values {
		ss = append(ss, e.key+"="+e.value)
	}
	return strings.Join(ss, ",")
}

func (o *keyToValue) Map() map[string]string {
	return o.mapped
}
