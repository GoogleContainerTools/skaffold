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

package docker

import (
	"github.com/spf13/cobra"
)

var (
	filename, dockerfile, context string
)

func AddDockerFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&filename, "filename", "f", "", "Filename or URL to the pipeline file")
	cmd.Flags().StringVarP(&dockerfile, "dockerfile", "d", "Dockerfile", "Dockerfile path")
	cmd.Flags().StringVarP(&context, "context", "c", "", "Dockerfile context path")
	cmd.Flags().VarP(depsFormatFlag, "output", "o", depsFormatFlag.Usage())
}
