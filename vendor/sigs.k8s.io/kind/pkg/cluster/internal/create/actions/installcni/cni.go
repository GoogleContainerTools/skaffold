/*
Copyright 2019 The Kubernetes Authors.

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

// Package installcni implements the install CNI action
package installcni

import (
	"bytes"
	"strings"
	"text/template"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/internal/apis/config"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/internal/patch"
)

type action struct{}

// NewAction returns a new action for installing default CNI
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Installing CNI 🔌")
	defer ctx.Status.End(false)

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// get the target node for this task
	controlPlanes, err := nodeutils.ControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	node := controlPlanes[0] // kind expects at least one always

	// read the manifest from the node
	var raw bytes.Buffer
	if err := node.Command("cat", "/kind/manifests/default-cni.yaml").SetStdout(&raw).Run(); err != nil {
		return errors.Wrap(err, "failed to read CNI manifest")
	}
	manifest := raw.String()

	// TODO: remove this check?
	// backwards compatibility for mounting your own manifest file to the default
	// location
	// NOTE: this is intentionally undocumented, as an internal implementation
	// detail. Going forward users should disable the default CNI and install
	// their own, or use the default. The internal templating mechanism is
	// not intended for external usage and is unstable.
	if strings.Contains(manifest, "would you kindly template this file") {
		t, err := template.New("cni-manifest").Parse(manifest)
		if err != nil {
			return errors.Wrap(err, "failed to parse CNI manifest template")
		}
		var out bytes.Buffer
		err = t.Execute(&out, &struct {
			PodSubnet string
		}{
			PodSubnet: ctx.Config.Networking.PodSubnet,
		})
		if err != nil {
			return errors.Wrap(err, "failed to execute CNI manifest template")
		}
		manifest = out.String()
	}

	// NOTE: this is intentionally undocumented, as an internal implementation
	// detail. Going forward users should disable the default CNI and install
	// their own, or use the default. The internal templating mechanism is
	// not intended for external usage and is unstable.
	if strings.Contains(manifest, "would you kindly patch this file") {
		// Add the controlplane endpoint so kindnet doesn´t have to wait for kube-proxy
		controlPlaneEndpoint, err := ctx.Provider.GetAPIServerInternalEndpoint(ctx.Config.Name)
		if err != nil {
			return err
		}

		patchValue := `
- op: add
  path: /spec/template/spec/containers/0/env/-
  value:
    name: CONTROL_PLANE_ENDPOINT
    value: ` + controlPlaneEndpoint

		controlPlanePatch6902 := config.PatchJSON6902{
			Group:   "apps",
			Version: "v1",
			Kind:    "DaemonSet",
			Patch:   patchValue,
		}

		patchedConfig, err := patch.KubeYAML(manifest, nil, []config.PatchJSON6902{controlPlanePatch6902})
		if err != nil {
			return err
		}
		manifest = patchedConfig
	}

	ctx.Logger.V(5).Infof("Using the following Kindnetd config:\n%s", manifest)

	// install the manifest
	if err := node.Command(
		"kubectl", "create", "--kubeconfig=/etc/kubernetes/admin.conf",
		"-f", "-",
	).SetStdin(strings.NewReader(manifest)).Run(); err != nil {
		return errors.Wrap(err, "failed to apply overlay network")
	}

	// mark success
	ctx.Status.End(true)
	return nil
}
