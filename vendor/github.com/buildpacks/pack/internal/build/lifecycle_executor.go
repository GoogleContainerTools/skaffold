package build

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/platform/files"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/container"
	"github.com/buildpacks/pack/pkg/cache"
	"github.com/buildpacks/pack/pkg/dist"
	"github.com/buildpacks/pack/pkg/logging"
)

var (
	// SupportedPlatformAPIVersions lists the Platform API versions pack supports listed from earliest to latest
	SupportedPlatformAPIVersions = builder.APISet{
		api.MustParse("0.3"),
		api.MustParse("0.4"),
		api.MustParse("0.5"),
		api.MustParse("0.6"),
		api.MustParse("0.7"),
		api.MustParse("0.8"),
		api.MustParse("0.9"),
		api.MustParse("0.10"),
		api.MustParse("0.11"),
		api.MustParse("0.12"),
	}
)

type Builder interface {
	Name() string
	UID() int
	GID() int
	LifecycleDescriptor() builder.LifecycleDescriptor
	Stack() builder.StackMetadata
	RunImages() []builder.RunImageMetadata
	Image() imgutil.Image
	OrderExtensions() dist.Order
}

type LifecycleExecutor struct {
	logger logging.Logger
	docker DockerClient
}

type Cache interface {
	Name() string
	Clear(context.Context) error
	Type() cache.Type
}

type Termui interface {
	logging.Logger

	Run(funk func()) error
	Handler() container.Handler
	ReadLayers(reader io.ReadCloser) error
}

type LifecycleOptions struct {
	AppPath              string
	Image                name.Reference
	Builder              Builder
	BuilderImage         string // differs from Builder.Name() and Builder.Image().Name() in that it includes the registry context
	LifecycleImage       string
	LifecycleApis        []string // optional - populated only if custom lifecycle image is downloaded, from that lifecycle's container's Labels.
	RunImage             string
	FetchRunImage        func(name string) error
	ProjectMetadata      files.ProjectMetadata
	ClearCache           bool
	Publish              bool
	TrustBuilder         bool
	UseCreator           bool
	Interactive          bool
	Layout               bool
	Termui               Termui
	DockerHost           string
	Cache                cache.CacheOpts
	CacheImage           string
	HTTPProxy            string
	HTTPSProxy           string
	NoProxy              string
	Network              string
	AdditionalTags       []string
	Volumes              []string
	DefaultProcessType   string
	FileFilter           func(string) bool
	Workspace            string
	GID                  int
	PreviousImage        string
	ReportDestinationDir string
	SBOMDestinationDir   string
	CreationTime         *time.Time
}

func NewLifecycleExecutor(logger logging.Logger, docker DockerClient) *LifecycleExecutor {
	return &LifecycleExecutor{logger: logger, docker: docker}
}

func (l *LifecycleExecutor) Execute(ctx context.Context, opts LifecycleOptions) error {
	tmpDir, err := os.MkdirTemp("", "pack.tmp")
	if err != nil {
		return err
	}

	lifecycleExec, err := NewLifecycleExecution(l.logger, l.docker, tmpDir, opts)
	if err != nil {
		return err
	}

	if !opts.Interactive {
		defer lifecycleExec.Cleanup()
		return lifecycleExec.Run(ctx, NewDefaultPhaseFactory)
	}

	return opts.Termui.Run(func() {
		defer lifecycleExec.Cleanup()
		lifecycleExec.Run(ctx, NewDefaultPhaseFactory)
	})
}
