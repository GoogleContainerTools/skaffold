package build

import (
	"context"
	"io"
	"math/rand"
	"time"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/cache"
	"github.com/buildpacks/pack/internal/container"
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
	}
)

type Builder interface {
	Name() string
	UID() int
	GID() int
	LifecycleDescriptor() builder.LifecycleDescriptor
	Stack() builder.StackMetadata
	Image() imgutil.Image
}

type LifecycleExecutor struct {
	logger logging.Logger
	docker client.CommonAPIClient
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

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

type LifecycleOptions struct {
	AppPath            string
	Image              name.Reference
	Builder            Builder
	LifecycleImage     string
	RunImage           string
	ProjectMetadata    platform.ProjectMetadata
	ClearCache         bool
	Publish            bool
	TrustBuilder       bool
	UseCreator         bool
	Interactive        bool
	Termui             Termui
	DockerHost         string
	CacheImage         string
	HTTPProxy          string
	HTTPSProxy         string
	NoProxy            string
	Network            string
	AdditionalTags     []string
	Volumes            []string
	DefaultProcessType string
	FileFilter         func(string) bool
	Workspace          string
	GID                int
	PreviousImage      string
	SBOMDestinationDir string
}

func NewLifecycleExecutor(logger logging.Logger, docker client.CommonAPIClient) *LifecycleExecutor {
	return &LifecycleExecutor{logger: logger, docker: docker}
}

func (l *LifecycleExecutor) Execute(ctx context.Context, opts LifecycleOptions) error {
	lifecycleExec, err := NewLifecycleExecution(l.logger, l.docker, opts)
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
