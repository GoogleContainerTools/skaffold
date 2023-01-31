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
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/transform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/render/validate"
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

	transformer        transform.Transformer
	validator          validate.Validator
	transformAllowlist map[apimachinery.GroupKind]latest.ResourceFilter
	transformDenylist  map[apimachinery.GroupKind]latest.ResourceFilter
}

func (k Kustomize) Render(ctx context.Context, out io.Writer, builds []graph.Artifact, offline bool) (manifest.ManifestListByConfig, error) {

	var manifests manifest.ManifestList
	kCLI := kubectl.NewCLI(k.cfg, "")
	useKubectlKustomize := !generate.KustomizeBinaryCheck() && generate.KubectlVersionCheck(kCLI)

	for _, kustomizePath := range k.rCfg.Kustomize.Paths {
		var out []byte
		var err error
		kPath, err := sUtil.ExpandEnvTemplate(kustomizePath, nil)
		if err != nil {
			return manifest.NewManifestListByConfig(), fmt.Errorf("unable to parse path %q: %w", kustomizePath, err)
		}

		temp, err := os.MkdirTemp("", "*")
		if err != nil {
			return manifest.NewManifestListByConfig(), err
		}
		fs := newTmpFS(temp)

		kptfns, err := k.transformer.GetDeclarativeTransformers()

		if err != nil {
			return manifest.NewManifestListByConfig(), err
		}

		if len(kptfns) > 0 {

			k.mirror(kPath, fs)
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

func (k Kustomize) mirror(kusDir string, fs TmpFS) error {
	kFile := filepath.Join(kusDir, constants.KustomizeFilePaths[0])
	bytes, err := ioutil.ReadFile(kFile)
	if err != nil {
		return err
	}

	if err := fs.WriteTo(kFile, bytes); err != nil {
		return err
	}
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
	if err := yaml.Unmarshal(bytes, &kustomization); err != nil {
		return err
	}
	if err := k.mirrorPatchesStrategicMerge(kusDir, fs, kustomization); err != nil {
		return err
	}
	if err := k.mirrorResources(kusDir, fs, kustomization); err != nil {
		return err
	}

	return nil

}

func (k Kustomize) mirrorPatchesStrategicMerge(kusDir string, fs TmpFS, kustomization types.Kustomization) error {
	for _, p := range kustomization.PatchesStrategicMerge {
		pFile := filepath.Join(kusDir, string(p))
		bytes, err := ioutil.ReadFile(pFile)
		if err := fs.WriteTo(pFile, bytes); err != nil {
			return err
		}
		path, err := fs.GetPath(pFile)

		if err != nil {
			return err
		}

		err = k.transformer.TransformPath(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorResources(kusDir string, fs TmpFS, kustomization types.Kustomization) error {
	for _, r := range kustomization.Resources {
		// note that r is relative to kustomization file not working dir here
		rPath := filepath.Join(kusDir, r)
		stat, err := os.Stat(rPath)
		if err != nil {
			fmt.Println(err)
			return err
		}
		if stat.IsDir() {
			err := k.mirror(rPath, fs)
			if err != nil {
				return err
			}
		} else {
			// copy to tmpRoot, relative kusDir
			rFile := rPath
			bytes, err := ioutil.ReadFile(rFile)
			if err := fs.WriteTo(rFile, bytes); err != nil {
				return err
			}
			path, err := fs.GetPath(rFile)

			err = k.transformer.TransformPath(path)
			if err != nil {
				return err
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

	var validator validate.Validator
	if rCfg.Validate != nil {
		validator, err = validate.NewValidator(*rCfg.Validate)
		if err != nil {
			return Kustomize{}, err
		}
	}

	var transformer transform.Transformer
	if rCfg.Transform != nil {
		transformer, err = transform.NewTransformer(*rCfg.Transform)
		if err != nil {
			return Kustomize{}, err
		}
	}

	if len(manifestOverrides) > 0 {
		err := transformer.Append(latest.Transformer{Name: "apply-setters", ConfigMap: sUtil.EnvMapToSlice(manifestOverrides, ":")})
		if err != nil {
			return Kustomize{}, err
		}
	}

	return Kustomize{
		cfg:               cfg,
		configName:        configName,
		namespace:         ns,
		labels:            labels,
		rCfg:              rCfg,
		manifestOverrides: manifestOverrides,
		validator:         validator,
		transformer:       transformer,

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
//		expandedKustomizePath, err := sUtil.ExpandEnvTemplate(kustomizePath, nil)
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
