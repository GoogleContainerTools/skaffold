package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	bldr "github.com/buildpacks/pack/internal/builder"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/cache"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/project"
	projectTypes "github.com/buildpacks/pack/pkg/project/types"
)

type BuildFlags struct {
	Publish              bool
	ClearCache           bool
	TrustBuilder         bool
	TrustExtraBuildpacks bool
	Interactive          bool
	Sparse               bool
	DockerHost           string
	CacheImage           string
	Cache                cache.CacheOpts
	AppPath              string
	Builder              string
	Registry             string
	RunImage             string
	Platform             string
	Policy               string
	Network              string
	DescriptorPath       string
	DefaultProcessType   string
	LifecycleImage       string
	Env                  []string
	EnvFiles             []string
	Buildpacks           []string
	Extensions           []string
	Volumes              []string
	AdditionalTags       []string
	Workspace            string
	GID                  int
	UID                  int
	PreviousImage        string
	SBOMDestinationDir   string
	ReportDestinationDir string
	DateTime             string
	PreBuildpacks        []string
	PostBuildpacks       []string
}

// Build an image from source code
func Build(logger logging.Logger, cfg config.Config, packClient PackClient) *cobra.Command {
	var flags BuildFlags

	cmd := &cobra.Command{
		Use:     "build <image-name>",
		Args:    cobra.ExactArgs(1),
		Short:   "Generate app image from source code",
		Example: "pack build test_img --path apps/test-app --builder cnbs/sample-builder:bionic",
		Long: "Pack Build uses Cloud Native Buildpacks to create a runnable app image from source code.\n\nPack Build " +
			"requires an image name, which will be generated from the source code. Build defaults to the current directory, " +
			"but you can use `--path` to specify another source code directory. Build requires a `builder`, which can either " +
			"be provided directly to build using `--builder`, or can be set using the `set-default-builder` command. For more " +
			"on how to use `pack build`, see: https://buildpacks.io/docs/app-developer-guide/build-an-app/.",
		RunE: logError(logger, func(cmd *cobra.Command, args []string) error {
			inputImageName := client.ParseInputImageReference(args[0])
			if err := validateBuildFlags(&flags, cfg, inputImageName, logger); err != nil {
				return err
			}

			inputPreviousImage := client.ParseInputImageReference(flags.PreviousImage)

			descriptor, actualDescriptorPath, err := parseProjectToml(flags.AppPath, flags.DescriptorPath, logger)
			if err != nil {
				return err
			}

			if actualDescriptorPath != "" {
				logger.Debugf("Using project descriptor located at %s", style.Symbol(actualDescriptorPath))
			}

			builder := flags.Builder
			// We only override the builder to the one in the project descriptor
			// if it was not explicitly set by the user
			if !cmd.Flags().Changed("builder") && descriptor.Build.Builder != "" {
				builder = descriptor.Build.Builder
			}

			if builder == "" {
				suggestSettingBuilder(logger, packClient)
				return client.NewSoftError()
			}

			buildpacks := flags.Buildpacks
			extensions := flags.Extensions

			env, err := parseEnv(flags.EnvFiles, flags.Env)
			if err != nil {
				return err
			}

			isTrusted, err := bldr.IsTrustedBuilder(cfg, builder)
			if err != nil {
				return err
			}
			trustBuilder := isTrusted || bldr.IsKnownTrustedBuilder(builder) || flags.TrustBuilder
			if trustBuilder {
				logger.Debugf("Builder %s is trusted", style.Symbol(builder))
				if flags.LifecycleImage != "" {
					logger.Warn("Ignoring the provided lifecycle image as the builder is trusted, running the creator in a single container using the provided builder")
				}
			} else {
				logger.Debugf("Builder %s is untrusted", style.Symbol(builder))
				logger.Debug("As a result, the phases of the lifecycle which require root access will be run in separate trusted ephemeral containers.")
				logger.Debug("For more information, see https://medium.com/buildpacks/faster-more-secure-builds-with-pack-0-11-0-4d0c633ca619")
			}

			if !trustBuilder && len(flags.Volumes) > 0 {
				logger.Warn("Using untrusted builder with volume mounts. If there is sensitive data in the volumes, this may present a security vulnerability.")
			}

			stringPolicy := flags.Policy
			if stringPolicy == "" {
				stringPolicy = cfg.PullPolicy
			}
			pullPolicy, err := image.ParsePullPolicy(stringPolicy)
			if err != nil {
				return errors.Wrapf(err, "parsing pull policy %s", flags.Policy)
			}

			var lifecycleImage string
			if flags.LifecycleImage != "" {
				ref, err := name.ParseReference(flags.LifecycleImage)
				if err != nil {
					return errors.Wrapf(err, "parsing lifecycle image %s", flags.LifecycleImage)
				}
				lifecycleImage = ref.Name()
			}

			err = isForbiddenTag(cfg, inputImageName.Name(), lifecycleImage, builder)
			if err != nil {
				return errors.Wrapf(err, "forbidden image name")
			}

			var gid = -1
			if cmd.Flags().Changed("gid") {
				gid = flags.GID
			}

			var uid = -1
			if cmd.Flags().Changed("uid") {
				uid = flags.UID
			}

			dateTime, err := parseTime(flags.DateTime)
			if err != nil {
				return errors.Wrapf(err, "parsing creation time %s", flags.DateTime)
			}
			if err := packClient.Build(cmd.Context(), client.BuildOptions{
				AppPath:           flags.AppPath,
				Builder:           builder,
				Registry:          flags.Registry,
				AdditionalMirrors: getMirrors(cfg),
				AdditionalTags:    flags.AdditionalTags,
				RunImage:          flags.RunImage,
				Env:               env,
				Image:             inputImageName.Name(),
				Publish:           flags.Publish,
				DockerHost:        flags.DockerHost,
				Platform:          flags.Platform,
				PullPolicy:        pullPolicy,
				ClearCache:        flags.ClearCache,
				TrustBuilder: func(string) bool {
					return trustBuilder
				},
				TrustExtraBuildpacks: flags.TrustExtraBuildpacks,
				Buildpacks:           buildpacks,
				Extensions:           extensions,
				ContainerConfig: client.ContainerConfig{
					Network: flags.Network,
					Volumes: flags.Volumes,
				},
				DefaultProcessType:       flags.DefaultProcessType,
				ProjectDescriptorBaseDir: filepath.Dir(actualDescriptorPath),
				ProjectDescriptor:        descriptor,
				Cache:                    flags.Cache,
				CacheImage:               flags.CacheImage,
				Workspace:                flags.Workspace,
				LifecycleImage:           lifecycleImage,
				GroupID:                  gid,
				UserID:                   uid,
				PreviousImage:            inputPreviousImage.Name(),
				Interactive:              flags.Interactive,
				SBOMDestinationDir:       flags.SBOMDestinationDir,
				ReportDestinationDir:     flags.ReportDestinationDir,
				CreationTime:             dateTime,
				PreBuildpacks:            flags.PreBuildpacks,
				PostBuildpacks:           flags.PostBuildpacks,
				LayoutConfig: &client.LayoutConfig{
					Sparse:             flags.Sparse,
					InputImage:         inputImageName,
					PreviousInputImage: inputPreviousImage,
					LayoutRepoDir:      cfg.LayoutRepositoryDir,
				},
			}); err != nil {
				return errors.Wrap(err, "failed to build")
			}
			logger.Infof("Successfully built image %s", style.Symbol(inputImageName.Name()))
			return nil
		}),
	}
	buildCommandFlags(cmd, &flags, cfg)
	AddHelpFlag(cmd, "build")
	return cmd
}

func parseTime(providedTime string) (*time.Time, error) {
	var parsedTime time.Time
	switch providedTime {
	case "":
		return nil, nil
	case "now":
		parsedTime = time.Now().UTC()
	default:
		intTime, err := strconv.ParseInt(providedTime, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "parsing unix timestamp")
		}
		parsedTime = time.Unix(intTime, 0).UTC()
	}
	return &parsedTime, nil
}

func buildCommandFlags(cmd *cobra.Command, buildFlags *BuildFlags, cfg config.Config) {
	cmd.Flags().StringVarP(&buildFlags.AppPath, "path", "p", "", "Path to app dir or zip-formatted file (defaults to current working directory)")
	cmd.Flags().StringSliceVarP(&buildFlags.Buildpacks, "buildpack", "b", nil, "Buildpack to use. One of:\n  a buildpack by id and version in the form of '<buildpack>@<version>',\n  path to a buildpack directory (not supported on Windows),\n  path/URL to a buildpack .tar or .tgz file, or\n  a packaged buildpack image name in the form of '<hostname>/<repo>[:<tag>]'"+stringSliceHelp("buildpack"))
	cmd.Flags().StringSliceVarP(&buildFlags.Extensions, "extension", "", nil, "Extension to use. One of:\n  an extension by id and version in the form of '<extension>@<version>',\n  path to an extension directory (not supported on Windows),\n  path/URL to an extension .tar or .tgz file, or\n  a packaged extension image name in the form of '<hostname>/<repo>[:<tag>]'"+stringSliceHelp("extension"))
	cmd.Flags().StringVarP(&buildFlags.Builder, "builder", "B", cfg.DefaultBuilder, "Builder image")
	cmd.Flags().Var(&buildFlags.Cache, "cache",
		`Cache options used to define cache techniques for build process.
- Cache as bind: 'type=<build/launch>;format=bind;source=<path to directory>'
- Cache as image (requires --publish): 'type=<build/launch>;format=image;name=<registry image name>'
- Cache as volume: 'type=<build/launch>;format=volume;[name=<volume name>]'
    - If no name is provided, a random name will be generated.
`)
	cmd.Flags().StringVar(&buildFlags.CacheImage, "cache-image", "", `Cache build layers in remote registry. Requires --publish`)
	cmd.Flags().BoolVar(&buildFlags.ClearCache, "clear-cache", false, "Clear image's associated cache before building")
	cmd.Flags().StringVar(&buildFlags.DateTime, "creation-time", "", "Desired create time in the output image config. Accepted values are Unix timestamps (e.g., '1641013200'), or 'now'. Platform API version must be at least 0.9 to use this feature.")
	cmd.Flags().StringVarP(&buildFlags.DescriptorPath, "descriptor", "d", "", "Path to the project descriptor file")
	cmd.Flags().StringVarP(&buildFlags.DefaultProcessType, "default-process", "D", "", `Set the default process type. (default "web")`)
	cmd.Flags().StringArrayVarP(&buildFlags.Env, "env", "e", []string{}, "Build-time environment variable, in the form 'VAR=VALUE' or 'VAR'.\nWhen using latter value-less form, value will be taken from current\n  environment at the time this command is executed.\nThis flag may be specified multiple times and will override\n  individual values defined by --env-file."+stringArrayHelp("env")+"\nNOTE: These are NOT available at image runtime.")
	cmd.Flags().StringArrayVar(&buildFlags.EnvFiles, "env-file", []string{}, "Build-time environment variables file\nOne variable per line, of the form 'VAR=VALUE' or 'VAR'\nWhen using latter value-less form, value will be taken from current\n  environment at the time this command is executed\nNOTE: These are NOT available at image runtime.\"")
	cmd.Flags().StringVar(&buildFlags.Network, "network", "", "Connect detect and build containers to network")
	cmd.Flags().StringArrayVar(&buildFlags.PreBuildpacks, "pre-buildpack", []string{}, "Buildpacks to prepend to the groups in the builder's order")
	cmd.Flags().StringArrayVar(&buildFlags.PostBuildpacks, "post-buildpack", []string{}, "Buildpacks to append to the groups in the builder's order")
	cmd.Flags().BoolVar(&buildFlags.Publish, "publish", false, "Publish the application image directly to the container registry specified in <image-name>, instead of the daemon. The run image must also reside in the registry.")
	cmd.Flags().StringVar(&buildFlags.DockerHost, "docker-host", "",
		`Address to docker daemon that will be exposed to the build container.
If not set (or set to empty string) the standard socket location will be used.
Special value 'inherit' may be used in which case DOCKER_HOST environment variable will be used.
This option may set DOCKER_HOST environment variable for the build container if needed.
`)
	cmd.Flags().StringVar(&buildFlags.LifecycleImage, "lifecycle-image", cfg.LifecycleImage, `Custom lifecycle image to use for analysis, restore, and export when builder is untrusted.`)
	cmd.Flags().StringVar(&buildFlags.Platform, "platform", "", `Platform to build on (e.g., "linux/amd64").`)
	cmd.Flags().StringVar(&buildFlags.Policy, "pull-policy", "", `Pull policy to use. Accepted values are always, never, and if-not-present. (default "always")`)
	cmd.Flags().StringVarP(&buildFlags.Registry, "buildpack-registry", "r", cfg.DefaultRegistryName, "Buildpack Registry by name")
	cmd.Flags().StringVar(&buildFlags.RunImage, "run-image", "", "Run image (defaults to default stack's run image)")
	cmd.Flags().StringSliceVarP(&buildFlags.AdditionalTags, "tag", "t", nil, "Additional tags to push the output image to.\nTags should be in the format 'image:tag' or 'repository/image:tag'."+stringSliceHelp("tag"))
	cmd.Flags().BoolVar(&buildFlags.TrustBuilder, "trust-builder", false, "Trust the provided builder.\nAll lifecycle phases will be run in a single container.\nFor more on trusted builders, and when to trust or untrust a builder, check out our docs here: https://buildpacks.io/docs/tools/pack/concepts/trusted_builders")
	cmd.Flags().BoolVar(&buildFlags.TrustExtraBuildpacks, "trust-extra-buildpacks", false, "Trust buildpacks that are provided in addition to the buildpacks on the builder")
	cmd.Flags().StringArrayVar(&buildFlags.Volumes, "volume", nil, "Mount host volume into the build container, in the form '<host path>:<target path>[:<options>]'.\n- 'host path': Name of the volume or absolute directory path to mount.\n- 'target path': The path where the file or directory is available in the container.\n- 'options' (default \"ro\"): An optional comma separated list of mount options.\n    - \"ro\", volume contents are read-only.\n    - \"rw\", volume contents are readable and writeable.\n    - \"volume-opt=<key>=<value>\", can be specified more than once, takes a key-value pair consisting of the option name and its value."+stringArrayHelp("volume"))
	cmd.Flags().StringVar(&buildFlags.Workspace, "workspace", "", "Location at which to mount the app dir in the build image")
	cmd.Flags().IntVar(&buildFlags.GID, "gid", 0, `Override GID of user's group in the stack's build and run images. The provided value must be a positive number`)
	cmd.Flags().IntVar(&buildFlags.UID, "uid", 0, `Override UID of user in the stack's build and run images. The provided value must be a positive number`)
	cmd.Flags().StringVar(&buildFlags.PreviousImage, "previous-image", "", "Set previous image to a particular tag reference, digest reference, or (when performing a daemon build) image ID")
	cmd.Flags().StringVar(&buildFlags.SBOMDestinationDir, "sbom-output-dir", "", "Path to export SBoM contents.\nOmitting the flag will yield no SBoM content.")
	cmd.Flags().StringVar(&buildFlags.ReportDestinationDir, "report-output-dir", "", "Path to export build report.toml.\nOmitting the flag yield no report file.")
	cmd.Flags().BoolVar(&buildFlags.Interactive, "interactive", false, "Launch a terminal UI to depict the build process")
	cmd.Flags().BoolVar(&buildFlags.Sparse, "sparse", false, "Use this flag to avoid saving on disk the run-image layers when the application image is exported to OCI layout format")
	if !cfg.Experimental {
		cmd.Flags().MarkHidden("interactive")
		cmd.Flags().MarkHidden("sparse")
	}
}

func validateBuildFlags(flags *BuildFlags, cfg config.Config, inputImageRef client.InputImageReference, logger logging.Logger) error {
	if flags.Registry != "" && !cfg.Experimental {
		return client.NewExperimentError("Support for buildpack registries is currently experimental.")
	}

	if flags.Cache.Launch.Format == cache.CacheImage {
		logger.Warn("cache definition: 'launch' cache in format 'image' is not supported.")
	}

	if flags.Cache.Build.Format == cache.CacheImage && flags.CacheImage != "" {
		return errors.New("'cache' flag with 'image' format cannot be used with 'cache-image' flag.")
	}

	if flags.Cache.Build.Format == cache.CacheImage && !flags.Publish {
		return errors.New("image cache format requires the 'publish' flag")
	}

	if flags.CacheImage != "" && !flags.Publish {
		return errors.New("cache-image flag requires the publish flag")
	}

	if flags.GID < 0 {
		return errors.New("gid flag must be in the range of 0-2147483647")
	}

	if flags.UID < 0 {
		return errors.New("uid flag must be in the range of 0-2147483647")
	}

	if flags.Interactive && !cfg.Experimental {
		return client.NewExperimentError("Interactive mode is currently experimental.")
	}

	if inputImageRef.Layout() && !cfg.Experimental {
		return client.NewExperimentError("Exporting to OCI layout is currently experimental.")
	}

	if _, err := os.Stat(inputImageRef.Name()); err == nil && flags.AppPath == "" {
		logger.Warnf("You are building an image named '%s'. If you mean it as an app directory path, run 'pack build <args> --path %s'",
			inputImageRef.Name(), inputImageRef.Name())
	}

	return nil
}

func parseEnv(envFiles []string, envVars []string) (map[string]string, error) {
	env := map[string]string{}

	for _, envFile := range envFiles {
		envFileVars, err := parseEnvFile(envFile)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse env file '%s'", envFile)
		}

		for k, v := range envFileVars {
			env[k] = v
		}
	}
	for _, envVar := range envVars {
		env = addEnvVar(env, envVar)
	}
	return env, nil
}

func parseEnvFile(filename string) (map[string]string, error) {
	out := make(map[string]string)
	f, err := os.ReadFile(filepath.Clean(filename))
	if err != nil {
		return nil, errors.Wrapf(err, "open %s", filename)
	}
	for _, line := range strings.Split(string(f), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = addEnvVar(out, line)
	}
	return out, nil
}

func addEnvVar(env map[string]string, item string) map[string]string {
	arr := strings.SplitN(item, "=", 2)
	if len(arr) > 1 {
		env[arr[0]] = arr[1]
	} else {
		env[arr[0]] = os.Getenv(arr[0])
	}
	return env
}

func parseProjectToml(appPath, descriptorPath string, logger logging.Logger) (projectTypes.Descriptor, string, error) {
	actualPath := descriptorPath
	computePath := descriptorPath == ""

	if computePath {
		actualPath = filepath.Join(appPath, "project.toml")
	}

	if _, err := os.Stat(actualPath); err != nil {
		if computePath {
			return projectTypes.Descriptor{}, "", nil
		}
		return projectTypes.Descriptor{}, "", errors.Wrap(err, "stat project descriptor")
	}

	descriptor, err := project.ReadProjectDescriptor(actualPath, logger)
	return descriptor, actualPath, err
}

func isForbiddenTag(cfg config.Config, input, lifecycle, builder string) error {
	inputImage, err := name.ParseReference(input)
	if err != nil {
		return errors.Wrapf(err, "invalid image name %s", input)
	}

	if builder != "" {
		builderImage, err := name.ParseReference(builder)
		if err != nil {
			return errors.Wrapf(err, "parsing builder image %s", builder)
		}
		if inputImage.Context().RepositoryStr() == builderImage.Context().RepositoryStr() {
			return fmt.Errorf("name must not match builder image name")
		}
	}

	if lifecycle != "" {
		lifecycleImage, err := name.ParseReference(lifecycle)
		if err != nil {
			return errors.Wrapf(err, "parsing lifecycle image %s", lifecycle)
		}
		if inputImage.Context().RepositoryStr() == lifecycleImage.Context().RepositoryStr() {
			return fmt.Errorf("name must not match lifecycle image name")
		}
	}

	trustedBuilders := getTrustedBuilders(cfg)
	for _, trustedBuilder := range trustedBuilders {
		builder, err := name.ParseReference(trustedBuilder)
		if err != nil {
			return err
		}
		if inputImage.Context().RepositoryStr() == builder.Context().RepositoryStr() {
			return fmt.Errorf("name must not match trusted builder name")
		}
	}

	defaultLifecycleImageRef, err := name.ParseReference(config.DefaultLifecycleImageRepo)
	if err != nil {
		return errors.Wrapf(err, "parsing default lifecycle image %s", config.DefaultLifecycleImageRepo)
	}

	if inputImage.Context().RepositoryStr() == defaultLifecycleImageRef.Context().RepositoryStr() {
		return fmt.Errorf("name must not match default lifecycle image name")
	}

	if cfg.DefaultBuilder != "" {
		defaultBuilderImage, err := name.ParseReference(cfg.DefaultBuilder)
		if err != nil {
			return errors.Wrapf(err, "parsing default builder %s", cfg.DefaultBuilder)
		}
		if inputImage.Context().RepositoryStr() == defaultBuilderImage.Context().RegistryStr() {
			return fmt.Errorf("name must not match default builder image name")
		}
	}

	return nil
}
