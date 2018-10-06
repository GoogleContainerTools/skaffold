package acr

import (
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

type Builder struct {
	*latest.AzureContainerBuild
}

// Creates a new builder with the Azure Container config
func NewBuilder(cfg *latest.AzureContainerBuild) *Builder {
	return &Builder{
		AzureContainerBuild: cfg,
	}
}

// Labels specific to Azure Container Build
func (b *Builder) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Builder: "azure-container-build",
	}
}
