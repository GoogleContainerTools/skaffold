package build

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

type BazelDependencyResolver struct{}

const sourceQuery = "kind('source file', deps('%s'))"

func (*BazelDependencyResolver) GetDependencies(a *config.Artifact) ([]string, error) {
	cmd := exec.Command("bazel", "query", fmt.Sprintf(sourceQuery, a.BazelArtifact.BuildTarget), "--noimplicit_deps", "--order_output=no")
	stdout, stderr, err := util.RunCommand(cmd, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "stdout: %s stderr: %s", stdout, stderr)
	}
	labels := strings.Split(string(stdout), "\n")
	var deps []string
	for _, l := range labels {
		if strings.HasPrefix(l, "@") {
			continue
		}
		if strings.HasPrefix(l, "//external") {
			continue
		}
		dep := strings.TrimPrefix(l, "//:")
		deps = append(deps, dep)
	}
	return deps, nil
}
