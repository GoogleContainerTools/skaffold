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
	"io"
	"io/ioutil"
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

	if err != nil {
		return err
	}
	kustomization := Kustomization{}
	if err := yaml.Unmarshal(bytes, &kustomization); err != nil {
		return err
	}
	if err := k.mirrorPatchesStrategicMerge(kusDir, fs, kustomization.PatchesStrategicMerge); err != nil {
		return err
	}
	if err := k.mirrorPatchesJson6902(kusDir, fs, kustomization.PatchesJson6902); err != nil {
		return err
	}
	if err := k.mirrorPatches(kusDir, fs, kustomization.Patches); err != nil {
		return err
	}
	if err := k.mirrorResources(kusDir, fs, kustomization.Resources); err != nil {
		return err
	}
	if err := k.mirrorCrds(kusDir, fs, kustomization.Crds); err != nil {
		return err
	}

	if err := k.mirrorBases(kusDir, fs, kustomization.Bases); err != nil {
		return err
	}
	if err := k.mirrorConfigurations(kusDir, fs, kustomization.Configurations); err != nil {
		return err
	}
	if err := k.mirrorGenerators(kusDir, fs, kustomization.Generators); err != nil {
		return err
	}
	if err := k.mirrorTransformers(kusDir, fs, kustomization.Transformers); err != nil {
		return err
	}
	if err := k.mirrorValidators(kusDir, fs, kustomization.Validators); err != nil {
		return err
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

func (k Kustomize) mirrorPatchesStrategicMerge(kusDir string, fs TmpFS, merges []PatchStrategicMerge) error {
	for _, p := range merges {
		if err := k.mirrorFile(kusDir, fs, string(p)); err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorResources(kusDir string, fs TmpFS, resources []string) error {
	for _, r := range resources {
		// note that r is relative to kustomization file not working dir here
		rPath := filepath.Join(kusDir, r)
		stat, err := os.Stat(rPath)
		if err != nil {
			return err
		}
		if stat.IsDir() {
			if err := k.mirror(rPath, fs); err != nil {
				return err
			}
		} else {
			if err := k.mirrorFile(kusDir, fs, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k Kustomize) mirrorFile(kusDir string, fs TmpFS, path string) error {
	// todo making everything absolute path
	pFile := filepath.Join(kusDir, path)
	bytes, err := ioutil.ReadFile(pFile)
	if err := fs.WriteTo(pFile, bytes); err != nil {
		return err
	}
	fsPath, err := fs.GetPath(pFile)

	if err != nil {
		return err
	}

	err = k.transformer.TransformPath(fsPath)
	if err != nil {
		return err
	}
	return nil
}

func (k Kustomize) mirrorFiles(kusDir string, fs TmpFS, paths []string) error {
	for _, path := range paths {
		err := k.mirrorFile(kusDir, fs, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorBases(kusDir string, fs TmpFS, bases []string) error {

	for _, b := range bases {
		if err := k.mirror(filepath.Join(kusDir, b), fs); err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorCrds(kusDir string, fs TmpFS, crds []string) error {
	return k.mirrorFiles(kusDir, fs, crds)
}

func (k Kustomize) mirrorConfigurations(kusDir string, fs TmpFS, configurations []string) error {
	return k.mirrorFiles(kusDir, fs, configurations)
}

func (k Kustomize) mirrorGenerators(kusDir string, fs TmpFS, generators []string) error {
	return k.mirrorFiles(kusDir, fs, generators)
}

func (k Kustomize) mirrorTransformers(kusDir string, fs TmpFS, transformers []string) error {
	return k.mirrorFiles(kusDir, fs, transformers)
}

func (k Kustomize) mirrorValidators(kusDir string, fs TmpFS, validators []string) error {
	return k.mirrorFiles(kusDir, fs, validators)
}

func (k Kustomize) mirrorPatches(dir string, fs TmpFS, patches []Patch) error {
	for _, patch := range patches {
		if err := k.mirrorFile(dir, fs, patch.Path); err != nil {
			return err
		}
	}
	return nil
}

func (k Kustomize) mirrorPatchesJson6902(dir string, fs TmpFS, patches []Patch) error {
	return k.mirrorPatches(dir, fs, patches)
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
