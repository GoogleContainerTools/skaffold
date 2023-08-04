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

// Package installstorage implements the an action to install a default
// storageclass
package installstorage

import (
	"bytes"
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
)

type action struct{}

// NewAction returns a new action for installing storage
func NewAction() actions.Action {
	return &action{}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Installing StorageClass 💾")
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

	// add the default storage class
	if err := addDefaultStorage(ctx.Logger, node); err != nil {
		return errors.Wrap(err, "failed to add default storage class")
	}

	// mark success
	ctx.Status.End(true)
	return nil
}

// legacy default storage class
// we need this for e2es (StatefulSet)
// newer kind images ship a storage driver manifest
const defaultStorageManifest = `# host-path based default storage class
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  namespace: kube-system
  name: standard
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: kubernetes.io/host-path`

func addDefaultStorage(logger log.Logger, controlPlane nodes.Node) error {
	// start with fallback default, and then try to get the newer kind node
	// storage manifest if present
	manifest := defaultStorageManifest
	var raw bytes.Buffer
	if err := controlPlane.Command("cat", "/kind/manifests/default-storage.yaml").SetStdout(&raw).Run(); err != nil {
		logger.Warn("Could not read storage manifest, falling back on old k8s.io/host-path default ...")
	} else {
		manifest = raw.String()
	}

	// apply the manifest
	in := strings.NewReader(manifest)
	cmd := controlPlane.Command(
		"kubectl",
		"--kubeconfig=/etc/kubernetes/admin.conf", "apply", "-f", "-",
	)
	cmd.SetStdin(in)
	return cmd.Run()
}
