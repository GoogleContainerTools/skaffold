/*
Copyright 2019 The Skaffold Authors

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

package util

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/buildpacks/lifecycle/cmd"
	"golang.org/x/sync/semaphore"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	k8s "k8s.io/client-go/kubernetes"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/initializer/prompt"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubernetes/manifest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/stringset"
	timeutil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/time"
)

var (
	confirmHydrationDirOverride = prompt.ConfirmHydrationDirOverride
)

// ApplyDefaultRepo applies the default repo to a given image tag.
func ApplyDefaultRepo(globalConfig string, defaultRepo *string, tag string) (string, error) {
	repo, err := config.GetDefaultRepo(globalConfig, defaultRepo)
	if err != nil {
		return "", fmt.Errorf("getting default repo: %w", err)
	}

	multiLevel, err := config.GetMultiLevelRepo(globalConfig)
	if err != nil {
		return "", fmt.Errorf("getting multi-level repo support: %w", err)
	}

	newTag, err := docker.SubstituteDefaultRepoIntoImage(repo, multiLevel, tag)
	if err != nil {
		return "", fmt.Errorf("applying default repo to %q: %w", tag, err)
	}

	return newTag, nil
}

// Update which images are logged, if the image is present in the provided deployer's artifacts.
func AddTagsToPodSelector(runnerBuilds []graph.Artifact, deployerArtifacts []graph.Artifact, podSelector *kubernetes.ImageList) {
	// This implementation is mostly picked from v1 for fixing log duplication issue when multiple deployers are used.
	// According to the original author "Each Deployer will be directly responsible for adding its deployed artifacts to the PodSelector
	// by cross-referencing them against the list of images parsed out of the set of manifests they each deploy". Each deploy should only
	// add its own deployed artifacts to the PodSelector to avoid duplicate logging when multi-deployers are used.
	// This implementation only streams logs for the intersection of runnerBuilds and deployerArtifacts images, not all images from a deployer
	// probably because at that time the team didn't want to stream logs from images not built by Skaffold, e.g. images from docker hub, but this
	// may change. The initial implementation was using imageName as map key for getting shared elements, this was ok as deployerArtifacts were
	// parsed out from skaffold config files in v1 and tag was not available if not specified. Now deployers don't own render responsibilities
	// anymore, instead callers pass rendered manifests to deployers, we can only parse artifacts from these rendered manifests. The imageName
	// from deployerArtifacts here has the default-repo value as prefix while the one from runnerBuilds doesn't. This discrepancy causes artifact.Tag
	// fail to add into podSelector, which leads to podWatchers fail to get events from pods. As tags are available in deployerArtifacts now, so using
	// tag as map key to get the shared elements.
	m := map[string]bool{}
	for _, a := range deployerArtifacts {
		m[a.Tag] = true
	}
	for _, artifact := range runnerBuilds {
		if _, ok := m[artifact.Tag]; ok {
			podSelector.Add(artifact.Tag)
		}
	}
}

func MockK8sClient(string) (k8s.Interface, error) {
	return fakekubeclientset.NewSimpleClientset(), nil
}

func ConsolidateNamespaces(original, new []string) []string {
	if len(new) == 0 {
		return original
	}
	namespaces := stringset.New()
	namespaces.Insert(append(original, new...)...)
	namespaces.Delete("") // if we have provided namespaces, remove the empty "default" namespace
	return namespaces.ToList()
}

// GetHydrationDir points to the directory where the manifest rendering happens. By default, it is set to "<WORKDIR>/.kpt-pipeline".
func GetHydrationDir(opts config.SkaffoldOptions, workingDir string, promptIfNeeded bool, isKptRendererOrDeployerUsed bool) (string, error) {
	var hydratedDir string
	var err error

	if !isKptRendererOrDeployerUsed {
		log.Entry(context.TODO()).Info("no kpt renderer or deployer found, skipping hydrated-dir creation")
		return "", nil
	}

	if opts.HydrationDir == constants.DefaultHydrationDir {
		hydratedDir = filepath.Join(workingDir, constants.DefaultHydrationDir)
		promptIfNeeded = false
	} else {
		hydratedDir = opts.HydrationDir
	}
	if hydratedDir, err = filepath.Abs(hydratedDir); err != nil {
		return "", err
	}

	if _, err := os.Stat(hydratedDir); os.IsNotExist(err) {
		log.Entry(context.TODO()).Infof("hydrated-dir does not exist, creating %v\n", hydratedDir)
		if err := os.MkdirAll(hydratedDir, os.ModePerm); err != nil {
			return "", err
		}
	} else if !isDirEmpty(hydratedDir) {
		if promptIfNeeded && !opts.AssumeYes {
			fmt.Println("you can skip this promp message with flag \"--assume-yes=true\"")
			if ok := confirmHydrationDirOverride(os.Stdin); !ok {
				cmd.Exit(nil)
			}
		}
	}
	log.Entry(context.TODO()).Infof("manifests hydration will take place in %v\n", hydratedDir)
	return hydratedDir, nil
}

func isDirEmpty(dir string) bool {
	f, _ := os.Open(dir)
	defer f.Close()
	_, err := f.Readdirnames(1)
	return err == io.EOF
}

// GroupVersionResource returns the first `GroupVersionResource` for the given `GroupVersionKind`.
func GroupVersionResource(disco discovery.DiscoveryInterface, gvk schema.GroupVersionKind) (bool, schema.GroupVersionResource, error) {
	resources, err := disco.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return false, schema.GroupVersionResource{}, fmt.Errorf("getting server resources for group version: %w", err)
	}

	for _, r := range resources.APIResources {
		if r.Kind == gvk.Kind {
			return r.Namespaced, schema.GroupVersionResource{
				Group:    gvk.Group,
				Version:  gvk.Version,
				Resource: r.Name,
			}, nil
		}
	}

	return false, schema.GroupVersionResource{}, fmt.Errorf("could not find resource for %s", gvk.String())
}

func GetManifestsFromHydratedManifests(ctx context.Context, hydratedManifests []string) (manifest.ManifestList, error) {
	var manifests manifest.ManifestList
	for _, path := range hydratedManifests {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening hydrated manifest at %s: %w", path, err)
		}
		defer f.Close()
		ms, err := manifest.Load(f)
		if err != nil {
			return nil, fmt.Errorf("parsing manifests file into manifest list object: %w", err)
		}
		manifests = append(manifests, ms...)
	}

	return manifests, nil
}

type tagErr struct {
	tag string
	err error
}

// ImageTags generates tags for a list of artifacts
func ImageTags(ctx context.Context, runCtx *runcontext.RunContext, tagger tag.Tagger, out io.Writer, artifacts []*latest.Artifact) (tag.ImageTags, error) {
	start := time.Now()
	maxWorkers := runtime.GOMAXPROCS(0)

	if len(artifacts) > 0 {
		output.Default.Fprintln(out, "Generating tags...")
	} else {
		output.Default.Fprintln(out, "No tags generated")
	}

	tagErrs := make([]chan tagErr, len(artifacts))

	// Use a weighted semaphore as a counting semaphore to limit the number of simultaneous taggers.
	sem := semaphore.NewWeighted(int64(maxWorkers))
	for i := range artifacts {
		tagErrs[i] = make(chan tagErr, 1)

		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}

		i := i
		go func() {
			defer sem.Release(1)
			_tag, err := tag.GenerateFullyQualifiedImageName(ctx, tagger, *artifacts[i])
			tagErrs[i] <- tagErr{tag: _tag, err: err}
		}()
	}

	imageTags := make(tag.ImageTags, len(artifacts))
	showWarning := false

	for i, artifact := range artifacts {
		imageName := artifact.ImageName
		out, ctx := output.WithEventContext(ctx, out, constants.Build, imageName)
		output.Default.Fprintf(out, " - %s -> ", imageName)

		select {
		case <-ctx.Done():
			return nil, context.Canceled

		case t := <-tagErrs[i]:
			if t.err != nil {
				log.Entry(ctx).Debug(t.err)
				log.Entry(ctx).Debug("Using a fall-back tagger")

				fallbackTag, err := tag.GenerateFullyQualifiedImageName(ctx, &tag.ChecksumTagger{}, *artifact)
				if err != nil {
					return nil, fmt.Errorf("generating checksum as fall-back tag for %q: %w", imageName, err)
				}

				t.tag = fallbackTag
				showWarning = true
			}

			_tag, err := ApplyDefaultRepo(runCtx.GlobalConfig(), runCtx.DefaultRepo(), t.tag)

			if err != nil {
				return nil, err
			}

			fmt.Fprintln(out, _tag)
			imageTags[imageName] = _tag
		}
	}

	if showWarning {
		output.Yellow.Fprintln(out, "Some taggers failed. Rerun with -vdebug for errors.")
	}

	log.Entry(ctx).Infoln("Tags generated in", timeutil.Humanize(time.Since(start)))
	return imageTags, nil
}
