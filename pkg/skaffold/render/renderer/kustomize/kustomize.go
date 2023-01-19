package kustomize

import (
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/generate"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	sUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"

	"io"
	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	var manifests manifest.ManifestList
	kCLI := kubectl.NewCLI(k.cfg, "")
	useKubectlKustomize := !generate.KustomizeBinaryCheck() && generate.KubectlVersionCheck(kCLI)
	//mutators, err := transform.NewTransformer(*k.rCfg.Transform)
	//if err != nil {
	//	return manifest.ManifestListByConfig{}, err
	//}
	//transformers, err := mutators.GetDeclarativeTransformers()

	for _, kustomizePath := range k.rCfg.Kustomize.Paths {
		var out []byte
		var err error
		kPath, err := sUtil.ExpandEnvTemplate(kustomizePath, nil)
		if err != nil {
			return manifest.NewManifestListByConfig(), fmt.Errorf("unable to parse path %q: %w", kustomizePath, err)
		}

		if useKubectlKustomize {
			out, err = kCLI.Kustomize(ctx, kustomizeBuildArgs(k.rCfg.Kustomize.BuildArgs, kPath))
		} else {
			cmd := exec.CommandContext(ctx, "kustomize", append([]string{"build"}, kustomizeBuildArgs(k.rCfg.Kustomize.BuildArgs, kPath)...)...)
			out, err = sUtil.RunCmdOut(ctx, cmd)
		}

		if len(out) == 0 {
			continue
		}
		manifests.Append(out)
	}

	opts := util.GenerateHydratedManifestsOptions{
		TransformAllowList:         k.transformAllowlist,
		TransformDenylist:          k.transformDenylist,
		EnablePlatformNodeAffinity: k.cfg.EnablePlatformNodeAffinityInRenderedManifests(),
		EnableGKEARMNodeToleration: k.cfg.EnableGKEARMNodeTolerationInRenderedManifests(),
		Offline:                    offline,
		KubeContext:                k.cfg.GetKubeContext(),
	}

	ns := k.namespace
	if k.cfg.GetKubeNamespace() != "" {
		ns = k.cfg.GetKubeNamespace()
	}
	util.BaseTransform(ctx, manifests, builds, opts, k.labels, ns)
	manifestListByConfig := manifest.NewManifestListByConfig()
	//.Add(k.configName, manifests), nil
	manifestListByConfig.Add(k.configName, manifests)

	return manifestListByConfig, nil

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

// kustomizeBuildArgs returns a list of build args to be passed to kustomize build.
func kustomizeBuildArgs(buildArgs []string, kustomizePath string) []string {
	var args []string

	if len(buildArgs) > 0 {
		for _, v := range buildArgs {
			parts := strings.Split(v, " ")
			args = append(args, parts...)
		}
	}

	if len(kustomizePath) > 0 {
		args = append(args, kustomizePath)
	}

	return args
}
