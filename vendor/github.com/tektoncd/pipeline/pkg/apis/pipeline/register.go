/*
Copyright 2019 The Tekton Authors

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

package pipeline

const (
	// GroupName is the Kubernetes resource group name for Pipeline types.
	GroupName = "tekton.dev"

	// TaskLabelKey is used as the label identifier for a task
	TaskLabelKey = "/task"

	// TaskRunLabelKey is used as the label identifier for a TaskRun
	TaskRunLabelKey = "/taskRun"

	// PipelineLabelKey is used as the label identifier for a Pipeline
	PipelineLabelKey = "/pipeline"

	// PipelineRunLabelKey is used as the label identifier for a PipelineRun
	PipelineRunLabelKey = "/pipelineRun"

	// PipelineRunLabelKey is used as the label identifier for a PipelineTask
	PipelineTaskLabelKey = "/pipelineTask"

	// ConditionCheck is used as the label identifier for a ConditionCheck
	ConditionCheckKey = "/conditionCheck"
)
