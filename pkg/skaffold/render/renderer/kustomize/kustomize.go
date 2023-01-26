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
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/kptfile"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/renderer/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	sUtil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"sigs.k8s.io/kustomize/pkg/types"

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

	labels            map[string]string
	manifestOverrides map[string]string

	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func (k Kustomize) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {

	var manifests manifest.ManifestList
	kCLI := kubectl.NewCLI(k.cfg, "")
	useKubectlKustomize := !generate.KustomizeBinaryCheck() && generate.KubectlVersionCheck(kCLI)
	var tra []latest.Transformer
	if k.rCfg.Transform != nil {
		tra = *k.rCfg.Transform
	}
	mutators, err := transform.NewTransformer(tra)
	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}
	transformers, err := mutators.GetDeclarativeTransformers()
	if len(k.manifestOverrides) > 0 {
		transformers = append(transformers, kptfile.Function{Image: "gcr.io/kpt-fn/apply-setters:unstable", ConfigMap: k.manifestOverrides})
	}

	if err != nil {
		return manifest.ManifestListByConfig{}, err
	}

	for _, kustomizePath := range k.rCfg.Kustomize.Paths {
		var out []byte
		var err error
		kPath, err := sUtil.ExpandEnvTemplate(kustomizePath, nil)
		if err != nil {
			return manifest.NewManifestListByConfig(), fmt.Errorf("unable to parse path %q: %w", kustomizePath, err)
		}

		temp, err := os.MkdirTemp("", "*")
		if transformers != nil {

			mirror(kPath, temp, transformers)
			kPath = filepath.Join(temp, kPath)
		}

		if err != nil {
			return manifest.ManifestListByConfig{}, err
		}

		if useKubectlKustomize {
			out, err = kCLI.Kustomize(ctx, kustomizeBuildArgs(k.rCfg.Kustomize.BuildArgs, kPath))
		} else {
			cmd := exec.CommandContext(ctx, "kustomize", append([]string{"build"}, kustomizeBuildArgs(k.rCfg.Kustomize.BuildArgs, kPath)...)...)
			out, err = sUtil.RunCmdOut(ctx, cmd)
		}
		if err != nil {
			return manifest.ManifestListByConfig{}, err
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

// todo wrap tmpRoot into a struct, the struct should provide WriteTo, Clean method
func mirror(kusDir string, tmpRoot string, transformers []kptfile.Function) error {
	kFile := filepath.Join(kusDir, constants.KustomizeFilePaths[0])
	dstPath := filepath.Join(tmpRoot, kusDir)
	os.MkdirAll(dstPath, os.ModePerm)

	copy(kFile, filepath.Join(dstPath, constants.KustomizeFilePaths[0]))

	bytes, err := ioutil.ReadFile(kFile)
	// todo Write a new Kustomization file model or use the one from the latest kustomize lib
	// PatchesStrategicMerge, relative kusDir
	// PatchesJson6902, relative kusDir
	// Resources,  relative kusDir
	// Crds
	// Bases, relative kusDir, url
	// Configurations

	if err != nil {
		return err
	}
	kustomization := types.Kustomization{}
	err = yaml.Unmarshal(bytes, &kustomization)
	for _, p := range kustomization.PatchesStrategicMerge {
		pFile := filepath.Join(kusDir, string(p))
		dir := filepath.Dir(pFile)
		pDir := filepath.Join(tmpRoot, dir)
		err := os.MkdirAll(pDir, os.ModePerm)
		if err != nil {
			fmt.Println("...." + err.Error())
		}
		copy(pFile, filepath.Join(tmpRoot, pFile))
		for _, transformer := range transformers {
			var kvs []string
			for key, value := range transformer.ConfigMap {
				kvs = append(kvs, fmt.Sprintf("%s=%s", key, value))
			}
			fmt.Println(kvs)
			fmt.Println(transformer.Image)
			args := []string{"fn", "eval", "-i", transformer.Image, filepath.Join(tmpRoot, pFile), "--"}
			args = append(args, kvs...)
			command := exec.Command("kpt", args...)
			err := command.Run()
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	for _, r := range kustomization.Resources {
		// note that r is relative to kustomization file not working dir here
		rPath := filepath.Join(kusDir, r)
		stat, err := os.Stat(rPath)
		if err != nil {
			fmt.Println(err)
			return err
		}
		if stat.IsDir() {
			mirror(rPath, tmpRoot, transformers)
		} else {
			// copy to tmpRoot, relative kusDir
			rFile := rPath
			dir := filepath.Dir(rFile)
			pDir := filepath.Join(tmpRoot, dir)
			err := os.MkdirAll(pDir, os.ModePerm)
			if err != nil {
				fmt.Println(err)
			}
			copy(rFile, filepath.Join(tmpRoot, rFile))

			for _, transformer := range transformers {
				var kvs []string
				for key, value := range transformer.ConfigMap {
					kvs = append(kvs, fmt.Sprintf("%s=%s", key, value))
				}
				args := []string{"fn", "eval", "-i", transformer.Image, filepath.Join(tmpRoot, rFile), "--"}
				args = append(args, kvs...)
				command := exec.Command("kpt", args...)
				err := command.Run()
				if err != nil {
					fmt.Println(err)
				}
			}

		}
	}

	return nil

}

func New(cfg render.Config, rCfg latest.RenderConfig, labels map[string]string, configName string, ns string, manifestOverrides map[string]string) (Kustomize, error) {
	transformAllowlist, transformDenylist, err := util.ConsolidateTransformConfiguration(cfg)
	if err != nil {
		return Kustomize{}, err
	}
	return Kustomize{
		cfg:               cfg,
		configName:        configName,
		namespace:         ns,
		labels:            labels,
		rCfg:              rCfg,
		manifestOverrides: manifestOverrides,

		transformAllowlist: transformAllowlist,
		transformDenylist:  transformDenylist,
	}, nil
}

func (k Kustomize) ManifestDeps() ([]string, error) {

	return []string{}, nil
	//return kustomizeDependencies(k.cfg.GetWorkingDir(), k.rCfg.Kustomize.Paths)

}

//func kustomizeDependencies(workdir string, paths []string) ([]string, error) {
//	deps := stringset.New()
//	for _, kustomizePath := range paths {
//		expandedKustomizePath, err := util.ExpandEnvTemplate(kustomizePath, nil)
//		if err != nil {
//			return nil, fmt.Errorf("unable to parse path %q: %w", kustomizePath, err)
//		}
//
//		if !filepath.IsAbs(expandedKustomizePath) {
//			expandedKustomizePath = filepath.Join(workdir, expandedKustomizePath)
//		}
//		depsForKustomization, err := DependenciesForKustomization(expandedKustomizePath)
//		if err != nil {
//			return nil, sErrors.NewError(err,
//				&proto.ActionableErr{
//					Message: err.Error(),
//					ErrCode: proto.StatusCode_DEPLOY_KUSTOMIZE_USER_ERR,
//				})
//		}
//		deps.Insert(depsForKustomization...)
//	}
//	return deps.ToList(), nil
//}

func copy(str, dst string) (err error) {
	input, err := os.ReadFile(str)
	if err != nil {
		return
	}

	err = os.WriteFile(dst, input, 0644)

	return
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
