package build

import (
	"context"
	"math/rand"
	"time"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/lifecycle/api"
	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/logging"
)

var (
	// SupportedPlatformAPIVersions lists the Platform API versions pack supports listed from earliest to latest
	SupportedPlatformAPIVersions = builder.APISet{
		api.MustParse("0.3"),
		api.MustParse("0.4"),
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
	ClearCache         bool
	Publish            bool
	TrustBuilder       bool
	UseCreator         bool
	HTTPProxy          string
	HTTPSProxy         string
	NoProxy            string
	Network            string
	Volumes            []string
	DefaultProcessType string
	FileFilter         func(string) bool
}

func NewLifecycleExecutor(logger logging.Logger, docker client.CommonAPIClient) *LifecycleExecutor {
	return &LifecycleExecutor{logger: logger, docker: docker}
}

func (l *LifecycleExecutor) Execute(ctx context.Context, opts LifecycleOptions) error {
	lifecycleExec, err := NewLifecycleExecution(l.logger, l.docker, opts)
	if err != nil {
		return err
	}
	defer lifecycleExec.Cleanup()
	return lifecycleExec.Run(ctx)
}
