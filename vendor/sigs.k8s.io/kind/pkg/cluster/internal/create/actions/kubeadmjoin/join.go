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

// Package kubeadmjoin implements the kubeadm join action
package kubeadmjoin

import (
	"strings"

	"sigs.k8s.io/kind/pkg/cluster/constants"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/internal/version"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
)

// Action implements action for creating the kubeadm join
// and deploying it on the bootstrap control-plane node.
type Action struct{}

// NewAction returns a new action for creating the kubeadm jion
func NewAction() actions.Action {
	return &Action{}
}

// Execute runs the action
func (a *Action) Execute(ctx *actions.ActionContext) error {
	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// join secondary control plane nodes if any
	secondaryControlPlanes, err := nodeutils.SecondaryControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	if len(secondaryControlPlanes) > 0 {
		if err := joinSecondaryControlPlanes(ctx, secondaryControlPlanes); err != nil {
			return err
		}
	}

	// then join worker nodes if any
	workers, err := nodeutils.SelectNodesByRole(allNodes, constants.WorkerNodeRoleValue)
	if err != nil {
		return err
	}
	if len(workers) > 0 {
		if err := joinWorkers(ctx, workers); err != nil {
			return err
		}
	}

	return nil
}

func joinSecondaryControlPlanes(
	ctx *actions.ActionContext,
	secondaryControlPlanes []nodes.Node,
) error {
	ctx.Status.Start("Joining more control-plane nodes 🎮")
	defer ctx.Status.End(false)

	// TODO(bentheelder): it's too bad we can't do this concurrently
	// (this is not safe currently)
	for _, node := range secondaryControlPlanes {
		node := node // capture loop variable
		if err := runKubeadmJoin(ctx.Logger, node); err != nil {
			return err
		}
	}

	ctx.Status.End(true)
	return nil
}

func joinWorkers(
	ctx *actions.ActionContext,
	workers []nodes.Node,
) error {
	ctx.Status.Start("Joining worker nodes 🚜")
	defer ctx.Status.End(false)

	// create the workers concurrently
	fns := []func() error{}
	for _, node := range workers {
		node := node // capture loop variable
		fns = append(fns, func() error {
			return runKubeadmJoin(ctx.Logger, node)
		})
	}
	if err := errors.UntilErrorConcurrent(fns); err != nil {
		return err
	}

	ctx.Status.End(true)
	return nil
}

// runKubeadmJoin executes kubeadm join command
func runKubeadmJoin(logger log.Logger, node nodes.Node) error {
	kubeVersionStr, err := nodeutils.KubeVersion(node)
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes version from node")
	}
	kubeVersion, err := version.ParseGeneric(kubeVersionStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse kubernetes version %q", kubeVersionStr)
	}

	args := []string{
		"join",
		// the join command uses the config file generated in a well known location
		"--config", "/kind/kubeadm.conf",
		// increase verbosity for debugging
		"--v=6",
	}
	// Newer versions set this in the config file.
	if kubeVersion.LessThan(version.MustParseSemantic("v1.23.0")) {
		// Skip preflight to avoid pulling images.
		// Kind pre-pulls images and preflight may conflict with that.
		args = append(args, "--skip-phases=preflight")
	}

	// run kubeadm join
	cmd := node.Command("kubeadm", args...)
	lines, err := exec.CombinedOutputLines(cmd)
	logger.V(3).Info(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to join node with kubeadm")
	}

	return nil
}
