package deploy

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
)

type ComposeDeployer struct {
	*v1alpha2.DeployConfig
	kubeContext string
}

func NewComposeDeployer(cfg *v1alpha2.DeployConfig, kubeContext string) *ComposeDeployer {
	return &ComposeDeployer{
		DeployConfig: cfg,
		kubeContext:  kubeContext,
	}
}

func (c *ComposeDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Build) error {
	// loader := compose.Compose{}
	// opt := kobject.ConvertOptions{}
	// kObj, err := loader.LoadFile([]string{c.ComposeDeploy.ComposeFile})
	// if err != nil {
	// 	return err
	// }
	// t := &kubernetes.Kubernetes{Opt: opt}
	// objs, err := t.Transform(kObj, opt)
	// fmt.Println(objs)
	// list := &api.List{}
	// // convert objects to versioned and add them to list
	// for _, object := range objs {
	// 	versionedObject, err := convertToVersion(object, v1.GroupVersion{})
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// 	list.Items = append(list.Items, versionedObject)

	// }
	// // version list itself
	// listVersion := v1.GroupVersion{Group: "", Version: "v1"}
	// convertedList, err := convertToVersion(list, listVersion)
	// if err != nil {
	// 	return err
	// }
	// data, err := yaml.Marshal(convertedList)
	// if err != nil {
	// 	return fmt.Errorf("error in marshalling the List: %v", err)
	// }
	return nil
}

func (c *ComposeDeployer) Cleanup(context.Context, io.Writer) error {
	return nil
}

func (c *ComposeDeployer) Dependencies() ([]string, error) {
	return []string{}, nil
}
