package phase

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layout/sparse"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/internal/extend"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform"
)

type Extender struct {
	AppDir       string // explicitly ignored by the Dockerfile applier, also the Dockefile build context
	ExtendedDir  string // output directory for extended image layers
	GeneratedDir string // input Dockerfiles are found here
	ImageRef     string // the image to extend
	LayersDir    string // explicitly ignored by the Dockerfile applier
	PlatformDir  string // explicitly ignored by the Dockerfile applier

	CacheTTL          time.Duration            // a platform input
	DockerfileApplier DockerfileApplier        // uses kaniko, BuildKit, or other to apply the provided Dockerfile to the provided image
	Extensions        []buildpack.GroupElement // extensions are ordered from group.toml

	PlatformAPI *api.Version
}

// DockerfileApplier given a base image and a `build.Dockerfile` or `run.Dockerfile` will apply it to the base image
// and return a new image, or an error if encountered.
//
//go:generate mockgen -package testmock -destination testmock/dockerfile_applier.go github.com/buildpacks/lifecycle/phase DockerfileApplier
type DockerfileApplier interface {
	ImageFor(reference string) (v1.Image, error)
	Apply(dockerfile extend.Dockerfile, toBaseImage v1.Image, withBuildOptions extend.Options, logger log.Logger) (v1.Image, error)
	Cleanup() error
}

// NewExtender constructs a new Extender by initializing services and reading the provided analyzed and group files
// to determine the image to extend and the extensions to use.
func (f *HermeticFactory) NewExtender(inputs platform.LifecycleInputs, dockerfileApplier DockerfileApplier, logger log.Logger) (*Extender, error) {
	extender := &Extender{
		AppDir:            inputs.AppDir,
		ExtendedDir:       inputs.ExtendedDir,
		GeneratedDir:      inputs.GeneratedDir,
		LayersDir:         inputs.LayersDir,
		PlatformDir:       inputs.PlatformDir,
		CacheTTL:          inputs.KanikoCacheTTL,
		DockerfileApplier: dockerfileApplier,
		PlatformAPI:       f.platformAPI,
	}
	var err error
	if extender.ImageRef, err = f.getExtendImageRef(inputs, logger); err != nil {
		return nil, err
	}
	if extender.Extensions, err = f.getExtensions(inputs.GroupPath, logger); err != nil {
		return nil, err
	}
	return extender, nil
}

func (f *HermeticFactory) getExtendImageRef(inputs platform.LifecycleInputs, logger log.Logger) (string, error) {
	analyzedMD, err := f.configHandler.ReadAnalyzed(inputs.AnalyzedPath, logger)
	if err != nil {
		return "", err
	}
	if inputs.ExtendKind == "build" && analyzedMD.BuildImage != nil {
		return analyzedMD.BuildImage.Reference, nil
	}
	if inputs.ExtendKind == "run" && analyzedMD.RunImage != nil {
		return analyzedMD.RunImage.Reference, nil
	}
	return "", nil
}

func (e *Extender) Extend(kind string, logger log.Logger) error {
	switch kind {
	case buildpack.DockerfileKindBuild:
		return e.extendBuild(logger)
	case buildpack.DockerfileKindRun:
		return e.extendRun(logger)
	default:
		return nil
	}
}

func (e *Extender) extendBuild(logger log.Logger) error {
	origBaseImage, err := e.DockerfileApplier.ImageFor(e.ImageRef)
	if err != nil {
		return fmt.Errorf("getting build image to extend: %w", err)
	}

	extendedImage, err := e.extend(buildpack.DockerfileKindBuild, origBaseImage, logger)
	if err != nil {
		return fmt.Errorf("extending build image: %w", err)
	}

	if err = setImageEnvVarsInCurrentContext(extendedImage); err != nil {
		return fmt.Errorf("setting environment variables from extended image in current context: %w", err)
	}
	return e.DockerfileApplier.Cleanup()
}

func setImageEnvVarsInCurrentContext(image v1.Image) error {
	configFile, err := image.ConfigFile()
	if err != nil || configFile == nil {
		return fmt.Errorf("getting config for extended image: %w", err)
	}
	for _, env := range configFile.Config.Env {
		parts := strings.Split(env, "=")
		if len(parts) != 2 {
			return fmt.Errorf("parsing env '%s': expected format 'key=value'", env)
		}
		if err := os.Setenv(parts[0], parts[1]); err != nil {
			return fmt.Errorf("setting env: %w", err)
		}
	}
	return nil
}

func (e *Extender) extendRun(logger log.Logger) error {
	origBaseImage, err := e.DockerfileApplier.ImageFor(e.ImageRef)
	if err != nil {
		return fmt.Errorf("getting run image to extend: %w", err)
	}

	origTopLayer, origNumLayers, err := topLayerDigest(origBaseImage, logger)
	if err != nil {
		return fmt.Errorf("getting original run image top layer: %w", err)
	}
	logger.Debugf("Original image top layer digest: %s", origTopLayer)

	extendedImage, err := e.extend(buildpack.DockerfileKindRun, origBaseImage, logger)
	if err != nil {
		return fmt.Errorf("extending run image: %w", err)
	}

	if err = e.saveSparse(extendedImage, origTopLayer, origNumLayers, logger); err != nil {
		return fmt.Errorf("failed to copy extended image to output directory: %w", err)
	}
	return e.DockerfileApplier.Cleanup()
}

func topLayerDigest(image v1.Image, logger log.Logger) (string, int, error) {
	imageHash, err := image.Digest()
	if err != nil {
		return "", -1, err
	}

	manifest, err := image.Manifest()
	if err != nil {
		return "", -1, fmt.Errorf("getting image manifest: %w", err)
	}

	allLayers := manifest.Layers
	logger.Debugf("Found %d layers in original image with digest: %s", len(allLayers), imageHash)

	if len(allLayers) == 0 {
		return "", 0, nil
	}

	layer := allLayers[len(allLayers)-1]
	return layer.Digest.String(), len(allLayers), nil
}

func (e *Extender) saveSparse(image v1.Image, origTopLayerHash string, origNumLayers int, logger log.Logger) error {
	// save sparse image (manifest and config)
	imageHash, err := image.Digest()
	if err != nil {
		return fmt.Errorf("getting image hash: %w", err)
	}
	toPath := filepath.Join(e.ExtendedDir, "run", imageHash.String())
	layoutImage, err := sparse.NewImage(toPath, image)
	if err != nil {
		return fmt.Errorf("failed to initialize image: %w", err)
	}
	if err := layoutImage.Save(); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	// copy only the extended layers (those following the original top layer) to the layout path
	// FIXME: it would be nice if this were supported natively in imgutil
	allLayers, err := image.Layers()
	if err != nil {
		return fmt.Errorf("getting image layers: %w", err)
	}
	logger.Debugf("Found %d layers in extended image with digest: %s", len(allLayers), imageHash)

	var (
		currentHash  v1.Hash
		needsCopying bool
	)
	if origTopLayerHash == "" { // if the original base image had no layers, copy all the layers
		needsCopying = true
	}
	group, _ := errgroup.WithContext(context.TODO())
	for idx, currentLayer := range allLayers {
		currentHash, err = currentLayer.Digest()
		if err != nil {
			return fmt.Errorf("getting layer hash: %w", err)
		}
		switch {
		case needsCopying:
			currentLayer := currentLayer // allow use in closure
			logger.Debugf("Copying layer with digest: %s", currentHash)
			group.Go(func() error {
				return copyLayer(currentLayer, toPath)
			})
		case currentHash.String() == origTopLayerHash && idx+1 == origNumLayers:
			logger.Debugf("Found original top layer with digest: %s", currentHash)
			needsCopying = true
			continue
		case currentHash.String() == origTopLayerHash:
			logger.Warnf("Original run image has duplicated top layer with digest: %s", currentHash)
			continue
		default:
			logger.Debugf("Skipping base layer with digest: %s", currentHash)
			continue
		}
	}
	return group.Wait()
}

func copyLayer(layer v1.Layer, toSparseImage string) error {
	digest, err := layer.Digest()
	if err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(toSparseImage, "blobs", digest.Algorithm, digest.Hex))
	if err != nil {
		return err
	}
	defer f.Close()
	rc, err := layer.Compressed()
	if err != nil {
		return err
	}
	defer rc.Close()
	_, err = io.Copy(f, rc)
	return err
}

const (
	argBuildID = "build_id"
	argUserID  = "user_id"
	argGroupID = "group_id"
)

func (e *Extender) extend(kind string, baseImage v1.Image, logger log.Logger) (v1.Image, error) {
	defer log.NewMeasurement("Extender", logger)()
	logger.Debugf("Extending base image for %s: %s", kind, e.ImageRef)
	dockerfiles, err := e.dockerfilesFor(kind, logger)
	if err != nil {
		return nil, fmt.Errorf("getting %s.Dockerfiles: %w", kind, err)
	}

	var (
		configFile     *v1.ConfigFile
		rebasable      = true // we don't require the initial base image to have io.buildpacks.rebasable=true
		workingHistory []v1.History
	)
	digest, err := baseImage.Digest()
	if err != nil {
		return nil, err
	}
	logger.Debugf("Original image has digest: %s", digest)

	// get config
	configFile, err = baseImage.ConfigFile()
	if err != nil {
		return nil, err
	}
	workingHistory = imgutil.NormalizedHistory(configFile.History, len(configFile.RootFS.DiffIDs))
	userID, groupID := userFrom(*configFile)
	origUserID := userID
	for _, dockerfile := range dockerfiles {
		buildOptions := e.extendOptions(dockerfile)
		dockerfile.Args = append([]extend.Arg{
			{Name: argBuildID, Value: uuid.New().String()},
			{Name: argUserID, Value: userID},
			{Name: argGroupID, Value: groupID},
		}, dockerfile.Args...)
		// apply Dockerfile
		if baseImage, err = e.DockerfileApplier.Apply(
			dockerfile,
			baseImage,
			buildOptions,
			logger,
		); err != nil {
			return nil, fmt.Errorf("applying Dockerfile to image: %w", err)
		}
		digest, err = baseImage.Digest()
		if err != nil {
			return nil, err
		}
		logger.Debugf("Intermediate image has digest: %s", digest)

		// update rebasable, history in config, and user/group IDs
		configFile, err = baseImage.ConfigFile()
		if err != nil || configFile == nil {
			return nil, fmt.Errorf("getting image config: %w", err)
		}
		// rebasable
		if !rebasable || !isRebasable(configFile) {
			rebasable = false
		}
		if configFile.Config.Labels == nil {
			configFile.Config.Labels = map[string]string{}
		}
		configFile.Config.Labels[RebasableLabel] = fmt.Sprintf("%t", rebasable)
		// history
		newHistory := imgutil.NormalizedHistory(configFile.History, len(configFile.RootFS.DiffIDs))
		for i := len(workingHistory); i < len(newHistory); i++ {
			workingHistory = append(
				workingHistory,
				v1.History{
					CreatedBy: fmt.Sprintf(layers.ExtensionLayerName, newHistory[i].CreatedBy, dockerfile.ExtensionID),
				},
			)
		}
		configFile.History = workingHistory
		prevUserID := userID
		userID, groupID = userFrom(*configFile)
		if isRoot(userID) {
			logger.Warnf("Extension from %s changed the user ID from %s to %s; this must not be the final user ID (a following extension must reset the user).", dockerfile.Path, prevUserID, userID)
		}
	}
	if isRoot(userID) && kind == "run" {
		return baseImage, fmt.Errorf("the final user ID is 0 (root); please add another extension that resets the user to non-root")
	}
	if userID != origUserID {
		logger.Warnf("The original user ID was %s but the final extension left the user ID set to %s.", origUserID, userID)
	}
	// build images don't mutate the config
	if kind == buildpack.DockerfileKindBuild {
		return baseImage, nil
	}
	// run images mutate the config
	baseImage, err = mutate.ConfigFile(baseImage, configFile)
	if err != nil {
		return nil, err
	}
	digest, err = baseImage.Digest()
	if err != nil {
		return nil, err
	}
	logger.Debugf("Final extended image has digest: %s", digest)
	return baseImage, nil
}

func userFrom(config v1.ConfigFile) (string, string) {
	user := strings.Split(config.Config.User, ":")
	if len(user) < 2 {
		return config.Config.User, ""
	}
	return user[0], user[1]
}

func isRoot(userID string) bool {
	return userID == "0" || userID == "root"
}

const RebasableLabel = "io.buildpacks.rebasable"

func isRebasable(config *v1.ConfigFile) bool {
	val, ok := config.Config.Labels[RebasableLabel]
	if !ok {
		// label unset
		return false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		// label not parsable
		return false
	}
	return b
}

func (e *Extender) dockerfilesFor(kind string, logger log.Logger) ([]extend.Dockerfile, error) {
	var dockerfiles []extend.Dockerfile
	for _, ext := range e.Extensions {
		dockerfile, err := e.dockerfileFor(kind, ext.ID)
		if err != nil {
			return nil, err
		}
		if dockerfile != nil {
			logger.Debugf("Found %s Dockerfile for extension '%s'", kind, ext.ID)
			dockerfiles = append(dockerfiles, *dockerfile)
		}
	}
	return dockerfiles, nil
}

func (e *Extender) dockerfileFor(kind, extID string) (*extend.Dockerfile, error) {
	var err error
	dockerfilePath := filepath.Join(e.GeneratedDir, kind, launch.EscapeID(extID), "Dockerfile")
	configPath := filepath.Join(e.GeneratedDir, kind, launch.EscapeID(extID), "extend-config.toml")
	contextDir := e.AppDir

	if e.PlatformAPI.AtLeast("0.13") {
		configPath = filepath.Join(e.GeneratedDir, launch.EscapeID(extID), "extend-config.toml")
		dockerfilePath = filepath.Join(e.GeneratedDir, launch.EscapeID(extID), fmt.Sprintf("%s.Dockerfile", kind))

		contextDir, err = e.contextDirFor(kind, extID)
		if err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(dockerfilePath); err != nil {
		return nil, nil
	}

	var config extend.Config
	_, err = toml.DecodeFile(configPath, &config)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var args []extend.Arg
	if kind == buildpack.DockerfileKindBuild {
		args = config.Build.Args
	} else {
		args = config.Run.Args
	}

	return &extend.Dockerfile{
		ExtensionID: extID,
		Path:        dockerfilePath,
		Args:        args,
		ContextDir:  contextDir,
	}, nil
}

func (e *Extender) contextDirFor(kind, extID string) (string, error) {
	sharedContextDir := filepath.Join(e.GeneratedDir, launch.EscapeID(extID), extend.SharedContextDir)
	kindContextDir := filepath.Join(e.GeneratedDir, launch.EscapeID(extID), fmt.Sprintf("%s.%s", extend.SharedContextDir, kind))

	for _, dir := range []string{kindContextDir, sharedContextDir} {
		if s, err := os.Stat(dir); err == nil && s.IsDir() {
			return dir, nil
		}
	}

	return e.AppDir, nil
}

func (e *Extender) extendOptions(dockerfile extend.Dockerfile) extend.Options {
	return extend.Options{
		BuildContext: dockerfile.ContextDir,
		CacheTTL:     e.CacheTTL,
		IgnorePaths:  []string{e.AppDir, e.LayersDir, e.PlatformDir},
	}
}
