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

package manifest

import (
	"fmt"

	apimachinery "k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/yaml"
)

const metadataField = "metadata"

type ResourceSelector interface {
	allowByGroupKind(apimachinery.GroupKind) bool
	allowByNavpath(apimachinery.GroupKind, string, string) (string, bool)
}

// TransformAllowlist is the default allowlist of kinds that can be transformed by Skaffold.
var TransformAllowlist = map[apimachinery.GroupKind]latest.ResourceFilter{
	{Group: "", Kind: "Pod"}: {
		GroupKind: "Pod",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "", Kind: "Service"}: {
		GroupKind: "Service",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "apps", Kind: "DaemonSet"}: {
		GroupKind: "DaemonSet.apps",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "apps", Kind: "Deployment"}: {
		GroupKind: "Deployment.apps",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "apps", Kind: "ReplicaSet"}: {
		GroupKind: "ReplicaSet.apps",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "apps", Kind: "StatefulSet"}: {
		GroupKind: "StatefulSet.apps",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "batch", Kind: "CronJob"}: {
		GroupKind: "CronJob.batch",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "batch", Kind: "Job"}: {
		GroupKind: "Job.batch",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "extensions", Kind: "DaemonSet"}: {
		GroupKind: "DaemonSet.extensions",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "extensions", Kind: "Deployment"}: {
		GroupKind: "Deployment.extensions",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	{Group: "extensions", Kind: "ReplicaSet"}: {
		GroupKind: "ReplicaSet.extensions",
		Image:     []string{".*"},
		Labels:    []string{".*"},
		PodSpec:   []string{".*"},
	},
	// TODO: Investigate exact requirements for adding `affinity` definitions for the following custom resource kinds
	{Group: "serving.knative.dev", Kind: "Service"}: {
		GroupKind: "Service.serving.knative.dev",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "agones.dev", Kind: "Fleet"}: {
		GroupKind: "Fleet.agones.dev",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "agones.dev", Kind: "GameServer"}: {
		GroupKind: "GameServer.agones.dev",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "argoproj.io", Kind: "Rollout"}: {
		GroupKind: "Rollout.argoproj.io",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "argoproj.io", Kind: "Workflow"}: {
		GroupKind: "Workflow.argoproj.io",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "argoproj.io", Kind: "CronWorkflow"}: {
		GroupKind: "CronWorkflow.argoproj.io",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "argoproj.io", Kind: "WorkflowTemplate"}: {
		GroupKind: "WorkflowTemplate.argoproj.io",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "argoproj.io", Kind: "ClusterWorkflowTemplate"}: {
		GroupKind: "ClusterWorkflowTemplate.argoproj.io",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "platform.confluent.io", Kind: "Connect"}: {
		GroupKind: "Connect.platform.confluent.io",
		Image:     []string{".spec.image.application", ".spec.image.init"},
		Labels:    []string{".*"},
	},
	{Group: "platform.confluent.io", Kind: "ControlCenter"}: {
		GroupKind: "ControlCenter.platform.confluent.io",
		Image:     []string{".spec.image.application", ".spec.image.init"},
		Labels:    []string{".*"},
	},
	{Group: "platform.confluent.io", Kind: "Kafka"}: {
		GroupKind: "Kafka.platform.confluent.io",
		Image:     []string{".spec.image.application", ".spec.image.init"},
		Labels:    []string{".*"},
	},
	{Group: "platform.confluent.io", Kind: "KsqlDB"}: {
		GroupKind: "KsqlDB.platform.confluent.io",
		Image:     []string{".spec.image.application", ".spec.image.init"},
		Labels:    []string{".*"},
	},
	{Group: "platform.confluent.io", Kind: "SchemaRegistry"}: {
		GroupKind: "SchemaRegistry.platform.confluent.io",
		Image:     []string{".spec.image.application", ".spec.image.init"},
		Labels:    []string{".*"},
	},
	{Group: "platform.confluent.io", Kind: "Zookeeper"}: {
		GroupKind: "Zookeeper.platform.confluent.io",
		Image:     []string{".spec.image.application", ".spec.image.init"},
		Labels:    []string{".*"},
	},
	{Group: "run.googleapis.com", Kind: "Job"}: {
		GroupKind: "Job.run.googleapis.com",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "run.googleapis.com", Kind: "WorkerPool"}: {
		GroupKind: "WorkerPool.run.googleapis.com",
		Image:     []string{".*"},
		Labels:    []string{".*"},
	},
	{Group: "kafka.strimzi.io", Kind: "KafkaConnect"}: {
		GroupKind: "KafkaConnect.kafka.strimzi.io",
		Image:     []string{".spec.image"},
		Labels:    []string{".*"},
	},
	{Group: "kafka.strimzi.io", Kind: "Kafka"}: {
		GroupKind: "Kafka.kafka.strimzi.io",
		Image:     []string{".spec.kafka.image", ".spec.zookeeper.image", ".spec.entityOperator.topicOperator.image", ".spec.entityOperator.userOperator.image", ".spec.kafkaExporter.image."},
		Labels:    []string{".*"},
	},
}

// TransformDenylist is the default denylist on the set of kinds that can be transformed by Skaffold.
var TransformDenylist = map[apimachinery.GroupKind]latest.ResourceFilter{
	{Group: "apps", Kind: "StatefulSet"}: {
		GroupKind: "StatefulSet.apps",
		Labels:    []string{".spec.volumeClaimTemplates.metadata.labels"},
	},
	{Group: "batch", Kind: "Job"}: {
		GroupKind: "Job.batch",
		Labels:    []string{".spec.template.metadata.labels"},
	},
}

// FieldVisitor represents the aggregation/transformation that should be performed on each traversed field.
type FieldVisitor interface {
	// Visit is called for each transformable key contained in the object and may apply transformations/aggregations on it.
	// It should return true to allow recursive traversal or false when the entry was transformed.
	Visit(gk apimachinery.GroupKind, navpath string, object map[string]interface{}, key string, value interface{}, rs ResourceSelector) bool
}

// Visit recursively visits all transformable object fields within the manifests and lets the visitor apply transformations/aggregations on them.
func (l *ManifestList) Visit(visitor FieldVisitor, rs ResourceSelector) (ManifestList, error) {
	var updated ManifestList

	for _, manifest := range *l {
		m := make(map[string]interface{})
		if err := yaml.Unmarshal(manifest, &m); err != nil {
			return nil, fmt.Errorf("reading Kubernetes YAML: %w", err)
		}

		if len(m) == 0 {
			continue
		}

		traverseManifestFields(m, visitor, rs)

		updatedManifest, err := yaml.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("marshalling yaml: %w", err)
		}

		updated = append(updated, updatedManifest)
	}

	return updated, nil
}

// traverseManifest traverses all transformable fields contained within the manifest.
func traverseManifestFields(manifest map[string]interface{}, visitor FieldVisitor, rs ResourceSelector) {
	var groupKind apimachinery.GroupKind
	var apiVersion string
	if value, ok := manifest["apiVersion"].(string); ok {
		apiVersion = value
	}
	var kind string
	if value, ok := manifest["kind"].(string); ok {
		kind = value
	}

	gvk := apimachinery.FromAPIVersionAndKind(apiVersion, kind)
	groupKind = apimachinery.GroupKind{
		Group: gvk.Group,
		Kind:  gvk.Kind,
	}

	if shouldTransformManifest(manifest, rs) {
		visitor = &recursiveVisitorDecorator{visitor}
	}
	visitFields(groupKind, "", manifest, visitor, rs)
}

func shouldTransformManifest(manifest map[string]interface{}, rs ResourceSelector) bool {
	var apiVersion string
	switch value := manifest["apiVersion"].(type) {
	case string:
		apiVersion = value
	default:
		return false
	}

	var kind string
	switch value := manifest["kind"].(type) {
	case string:
		kind = value
	default:
		return false
	}

	gvk := apimachinery.FromAPIVersionAndKind(apiVersion, kind)
	groupKind := apimachinery.GroupKind{
		Group: gvk.Group,
		Kind:  gvk.Kind,
	}

	if rs.allowByGroupKind(groupKind) {
		return true
	}

	for _, w := range ConfigConnectorResourceSelector {
		if w.Matches(gvk.Group, gvk.Kind) {
			return true
		}
	}

	return false
}

// recursiveVisitorDecorator adds recursion to a FieldVisitor.
type recursiveVisitorDecorator struct {
	delegate FieldVisitor
}

func (d *recursiveVisitorDecorator) Visit(gk apimachinery.GroupKind, navpath string, o map[string]interface{}, k string, v interface{}, rs ResourceSelector) bool {
	if d.delegate.Visit(gk, navpath, o, k, v, rs) {
		visitFields(gk, navpath, v, d, rs)
	}
	return false
}

// visitFields traverses all fields and calls the visitor for each.
// navpath: a '.' delimited path representing the fields navigated to this point
func visitFields(gk apimachinery.GroupKind, navpath string, o interface{}, visitor FieldVisitor, rs ResourceSelector) {
	switch entries := o.(type) {
	case []interface{}:
		for _, v := range entries {
			// this case covers lists so we don't update the navpath
			visitFields(gk, navpath, v, visitor, rs)
		}
	case map[string]interface{}:
		for k, v := range entries {
			visitor.Visit(gk, navpath+"."+k, entries, k, v, rs)
		}
	}
}
