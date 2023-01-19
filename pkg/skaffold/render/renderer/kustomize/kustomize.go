package kustomize

import (
	"context"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"io"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"path/filepath"
)

type Kustomize struct {
	cfg  render.Config
	rCfg latest.RenderConfig

	configName string
	namespace  string

	labels    map[string]string
	overrides map[string]string

	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func (k *Kustomize) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {

	var localPaths []string

	for _, p := range k.rCfg.Kustomize.Paths {
		path, err := util.ExpandEnvTemplate(p, nil)
		if err != nil {
			return manifest.NewManifestListByConfig(), err
		}
		localPaths = append(localPaths, path)
	}

	paths, _ := util.ExpandPathsGlob(k.cfg.GetWorkingDir(), localPaths)

}

// isKustomizeDir copied from generate.go
func isKustomizeDir(path string) (string, bool) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	var dir string
	switch mode := fileInfo.Mode(); {
	case mode.IsDir():
		dir = path
	case mode.IsRegular():
		dir = filepath.Dir(path)
	}

	for _, base := range constants.KustomizeFilePaths {
		if _, err := os.Stat(filepath.Join(dir, base)); os.IsNotExist(err) {
			continue
		}
		return dir, true
	}
	return "", false
}
