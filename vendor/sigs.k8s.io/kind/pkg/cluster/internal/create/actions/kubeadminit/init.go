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

// Package kubeadminit implements the kubeadm init action
package kubeadminit

import (
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"

	"sigs.k8s.io/kind/pkg/cluster/nodeutils"

	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/version"
)

// kubeadmInitAction implements action for executing the kubeadm init
// and a set of default post init operations like e.g. install the
// CNI network plugin.
type action struct {
	skipKubeProxy bool
}

// NewAction returns a new action for kubeadm init
func NewAction(cfg *config.Cluster) actions.Action {
	return &action{skipKubeProxy: cfg.Networking.KubeProxyMode == config.NoneProxyMode}
}

// Execute runs the action
func (a *action) Execute(ctx *actions.ActionContext) error {
	ctx.Status.Start("Starting control-plane 🕹️")
	defer ctx.Status.End(false)

	allNodes, err := ctx.Nodes()
	if err != nil {
		return err
	}

	// get the target node for this task
	// TODO: eliminate the concept of bootstrapcontrolplane node entirely
	// outside this method
	node, err := nodeutils.BootstrapControlPlaneNode(allNodes)
	if err != nil {
		return err
	}

	kubeVersionStr, err := nodeutils.KubeVersion(node)
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes version from node")
	}
	kubeVersion, err := version.ParseGeneric(kubeVersionStr)
	if err != nil {
		return errors.Wrapf(err, "failed to parse kubernetes version %q", kubeVersionStr)
	}

	args := []string{
		// init because this is the control plane node
		"init",
		// specify our generated config file
		"--config=/kind/kubeadm.conf",
		"--skip-token-print",
		// increase verbosity for debugging
		"--v=6",
	}

	// Newer versions set this in the config file.
	if kubeVersion.LessThan(version.MustParseSemantic("v1.23.0")) {
		// Skip preflight to avoid pulling images.
		// Kind pre-pulls images and preflight may conflict with that.
		skipPhases := "preflight"
		if a.skipKubeProxy {
			skipPhases += ",addon/kube-proxy"
		}
		args = append(args, "--skip-phases="+skipPhases)
	}

	// run kubeadm
	cmd := node.Command("kubeadm", args...)
	lines, err := exec.CombinedOutputLines(cmd)
	ctx.Logger.V(3).Info(strings.Join(lines, "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to init node with kubeadm")
	}

	// copy some files to the other control plane nodes
	otherControlPlanes, err := nodeutils.SecondaryControlPlaneNodes(allNodes)
	if err != nil {
		return err
	}
	for _, otherNode := range otherControlPlanes {
		for _, file := range []string{
			// copy over admin config so we can use any control plane to get it later
			"/etc/kubernetes/admin.conf",
			// copy over certs
			"/etc/kubernetes/pki/ca.crt", "/etc/kubernetes/pki/ca.key",
			"/etc/kubernetes/pki/front-proxy-ca.crt", "/etc/kubernetes/pki/front-proxy-ca.key",
			"/etc/kubernetes/pki/sa.pub", "/etc/kubernetes/pki/sa.key",
			// TODO: if we gain external etcd support these will be
			// handled differently
			"/etc/kubernetes/pki/etcd/ca.crt", "/etc/kubernetes/pki/etcd/ca.key",
		} {
			if err := nodeutils.CopyNodeToNode(node, otherNode, file); err != nil {
				return errors.Wrap(err, "failed to copy admin kubeconfig")
			}
		}
	}

	// if we are only provisioning one node, remove the control plane taint
	// https://kubernetes.io/docs/setup/independent/create-cluster-kubeadm/#master-isolation
	if len(allNodes) == 1 {
		// TODO: Once kubeadm 1.23 is no longer supported remove the <1.24 handling.
		// TODO: Once kubeadm 1.24 is no longer supported remove the <1.25 handling.
		// https://github.com/kubernetes-sigs/kind/issues/1699
		rawVersion, err := nodeutils.KubeVersion(node)
		if err != nil {
			return errors.Wrap(err, "failed to get Kubernetes version from node")
		}
		kubeVersion, err := version.ParseSemantic(rawVersion)
		if err != nil {
			return errors.Wrap(err, "could not parse Kubernetes version")
		}
		var taints []string
		if kubeVersion.LessThan(version.MustParseSemantic("v1.24.0-alpha.1.592+370031cadac624")) {
			// for versions older than 1.24 prerelease remove only the old taint
			taints = []string{"node-role.kubernetes.io/master-"}
		} else if kubeVersion.LessThan(version.MustParseSemantic("v1.25.0-alpha.0.557+84c8afeba39ec9")) {
			// for versions between 1.24 and 1.25 prerelease remove both the old and new taint
			taints = []string{"node-role.kubernetes.io/control-plane-", "node-role.kubernetes.io/master-"}
		} else {
			// for any newer version only remove the new taint
			taints = []string{"node-role.kubernetes.io/control-plane-"}
		}
		taintArgs := []string{"--kubeconfig=/etc/kubernetes/admin.conf", "taint", "nodes", "--all"}
		taintArgs = append(taintArgs, taints...)

		if err := node.Command(
			"kubectl", taintArgs...,
		).Run(); err != nil {
			return errors.Wrap(err, "failed to remove control plane taint")
		}
	}

	// Kubeadm will add `node.kubernetes.io/exclude-from-external-load-balancers` on control plane nodes.
	// For single node clusters, this means we cannot have a load balancer at all (MetalLB, etc), so remove the label.
	if len(allNodes) == 1 {
		labelArgs := []string{"--kubeconfig=/etc/kubernetes/admin.conf", "label", "nodes", "--all", "node.kubernetes.io/exclude-from-external-load-balancers-"}
		if err := node.Command(
			"kubectl", labelArgs...,
		).Run(); err != nil {
			return errors.Wrap(err, "failed to remove control plane load balancer label")
		}
	}

	// mark success
	ctx.Status.End(true)
	return nil
}
