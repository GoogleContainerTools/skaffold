/*
Copyright 2024 The Skaffold Authors

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

package tofu

import (
	"context"
	"fmt"
	"io"

	deploy "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/instrumentation"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tofu"
)

// CLI holds parameters to run tofu.
type CLI struct {
	*tofu.CLI
}

type Config interface {
	tofu.Config
	deploy.Config
}

func NewCLI(cfg Config) CLI {
	return CLI{
		CLI: tofu.NewCLI(cfg),
	}
}

// Apply runs `tofu apply` on a Workspace.
func (c *CLI) Apply(ctx context.Context, out io.Writer) error {
	ctx, endTrace := instrumentation.StartTrace(ctx, "Apply", map[string]string{
		"AppliedBy": "tofu",
	})
	defer endTrace()

	r, w := io.Pipe()
	w.Close()
	if err := c.Run(ctx, r, out, "apply", "-auto-approve"); err != nil {
		endTrace(instrumentation.TraceEndError(err))
		return fmt.Errorf("tofu apply: %w", err)
	}

	return nil
}
