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

package kubectl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	pkgkubectl "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// CLI holds parameters to run kubectl.
type CLI struct {
	*pkgkubectl.CLI
	Flags latest.KubectlFlags

	forceDeploy      bool
	waitForDeletions config.WaitForDeletions
	previousApply    ManifestList
}

// Resource describes a deployed resource.
type Resource struct {
	APIVersion string
	Kind       string
	Namespace  string
	Name       string
	UID        string
}

// Resources describes a list of deployed resources.
type Resources []Resource

func NewCLI(runCtx *runcontext.RunContext, flags latest.KubectlFlags) CLI {
	return CLI{
		CLI:              pkgkubectl.NewFromRunContext(runCtx),
		Flags:            flags,
		forceDeploy:      runCtx.ForceDeploy(),
		waitForDeletions: runCtx.WaitForDeletions(),
	}
}

// Delete runs `kubectl delete` on a list of manifests.
func (c *CLI) Delete(ctx context.Context, out io.Writer, manifests ManifestList) error {
	args := c.args(c.Flags.Delete, "--ignore-not-found=true", "-f", "-")
	if err := c.Run(ctx, manifests.Reader(), out, "delete", args...); err != nil {
		return fmt.Errorf("kubectl delete: %w", err)
	}

	return nil
}

// Apply runs `kubectl apply` on a list of manifests.
func (c *CLI) Apply(ctx context.Context, out io.Writer, manifests ManifestList) (Resources, error) {
	// Only redeploy modified or new manifests
	// TODO(dgageot): should we delete a manifest that was deployed and is not anymore?
	updated := c.previousApply.Diff(manifests)
	logrus.Debugln(len(manifests), "manifests to deploy.", len(updated), "are updated or new")
	c.previousApply = manifests
	if len(updated) == 0 {
		return nil, nil
	}

	args := []string{"-f", "-", "-ojson"}
	if c.forceDeploy {
		args = append(args, "--force", "--grace-period=0")
	}

	if c.Flags.DisableValidation {
		args = append(args, "--validate=false")
	}

	buf, err := c.RunOutInput(ctx, updated.Reader(), "apply", c.args(c.Flags.Apply, args...)...)
	if err != nil {
		return nil, fmt.Errorf("kubectl apply: %w", err)
	}

	result, err := parseJSONOutput(buf)
	if err != nil {
		return nil, err
	}

	var resources Resources
	for _, item := range result.Items {
		resources = append(resources, Resource{
			APIVersion: item.APIVersion,
			Kind:       item.Kind,
			Namespace:  item.Metadata.Name,
			Name:       item.Metadata.Name,
			UID:        item.Metadata.UID,
		})
	}

	return resources, nil
}

// WaitForDeletions waits for resource marked for deletion to complete their deletion.
func (c *CLI) WaitForDeletions(ctx context.Context, out io.Writer, manifests ManifestList) error {
	if !c.waitForDeletions.Enabled {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.waitForDeletions.Max)
	defer cancel()

	previousList := ""
	previousCount := 0

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%d resources failed to complete their deletion before a new deployment: %s", previousCount, previousList)
		default:
			// List resources in json format.
			buf, err := c.RunOutInput(ctx, manifests.Reader(), "get", c.args(nil, "-f", "-", "--ignore-not-found", "-ojson")...)
			if err != nil {
				return err
			}

			result, err := parseJSONOutput(buf)
			if err != nil {
				return err
			}

			// Find which ones are marked for deletion. They have a `metadata.deletionTimestamp` field.
			var marked []string
			for _, item := range result.Items {
				if item.Metadata.DeletionTimestamp != "" {
					marked = append(marked, item.Metadata.Name)
				}
			}
			if len(marked) == 0 {
				return nil
			}

			list := `"` + strings.Join(marked, `", "`) + `"`
			logrus.Debugln("Resources are marked for deletion:", list)
			if list != previousList {
				if len(marked) == 1 {
					fmt.Fprintf(out, "%s is marked for deletion, waiting for completion\n", list)
				} else {
					fmt.Fprintf(out, "%d resources are marked for deletion, waiting for completion: %s\n", len(marked), list)
				}

				previousList = list
				previousCount = len(marked)
			}

			select {
			case <-ctx.Done():
			case <-time.After(c.waitForDeletions.Delay):
			}
		}
	}
}

// ReadManifests reads a list of manifests in yaml format.
func (c *CLI) ReadManifests(ctx context.Context, manifests []string) (ManifestList, error) {
	var list []string
	for _, manifest := range manifests {
		list = append(list, "-f", manifest)
	}

	var dryRun = "--dry-run"
	compTo1_18, err := c.CLI.CompareVersionTo(ctx, 1, 18)
	if err != nil {
		return nil, err
	}
	if compTo1_18 >= 0 {
		dryRun += "=client"
	}

	args := c.args([]string{dryRun, "-oyaml"}, list...)
	if c.Flags.DisableValidation {
		args = append(args, "--validate=false")
	}

	buf, err := c.RunOut(ctx, "create", args...)
	if err != nil {
		return nil, fmt.Errorf("kubectl create: %w", err)
	}

	var manifestList ManifestList
	manifestList.Append(buf)

	return manifestList, nil
}

func (c *CLI) args(commandFlags []string, additionalArgs ...string) []string {
	args := make([]string, 0, len(c.Flags.Global)+len(commandFlags)+len(additionalArgs))

	args = append(args, c.Flags.Global...)
	args = append(args, commandFlags...)
	args = append(args, additionalArgs...)

	return args
}

func (r *Resources) Namespaces() []string {
	var namespaces []string

	for _, resource := range *r {
		namespaces = append(namespaces, resource.Namespace)
	}

	return namespaces
}

type list struct {
	Items []item `json:"items"`
}

type item struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name              string `json:"name"`
		UID               string `json:"uid"`
		DeletionTimestamp string `json:"deletionTimestamp"`
	} `json:"metadata"`
}

func parseJSONOutput(buf []byte) (list, error) {
	// No resource found.
	if len(buf) == 0 {
		return list{}, nil
	}

	// First, try to parse a list of items.
	var l list
	if err := json.Unmarshal(buf, &l); err != nil {
		return list{}, err
	}

	if len(l.Items) > 0 {
		return l, nil
	}

	// It could also be a single item.
	var i item
	if err := json.Unmarshal(buf, &i); err != nil {
		return list{}, err
	}

	return list{
		Items: []item{i},
	}, nil
}
