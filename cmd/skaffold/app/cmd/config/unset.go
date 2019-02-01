/*
Copyright 2018 The Skaffold Authors

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

package config

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func NewCmdUnset(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset",
		Short: "Unset a value in the global Skaffold config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveKubectlContext()
			if err := unsetConfigValue(args[0]); err != nil {
				return err
			}
			logUnsetConfigForUser(out, args[0])
			return nil
		},
	}
	AddConfigFlags(cmd)
	AddSetFlags(cmd)
	return cmd
}

func logUnsetConfigForUser(out io.Writer, key string) {
	if global {
		out.Write([]byte(fmt.Sprintf("unset global value %s", key)))
	} else {
		out.Write([]byte(fmt.Sprintf("unset value %s for context %s\n", key, kubecontext)))
	}
}

func unsetConfigValue(name string) error {
	return setConfigValue(name, "")
}
