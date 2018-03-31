package build

import "github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/config"

type BazelDependencyResolver struct{}

// TODO(r2d4): implement
func (*BazelDependencyResolver) GetDependencies(a *config.Artifact) ([]string, error) {
	return []string{}, nil
}
