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

package integrationtest

import (
	"bytes"
	"context"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func IntegrationTest(ctx context.Context, defaultLabeller *deploy.DefaultLabeller, runCtx *runcontext.RunContext) (string, error) {
	client, err := kubernetesutil.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "getting kubernetes client")
	}
	kubeCtl := kubectl.NewFromRunContext(runCtx)

	podSelector := runCtx.Cfg.IntegrationTest.PodSelector

	if len(podSelector) == 0 {
		return "", errors.Errorf("no PodSelector defined")
	}

	pod, err := getPod(client, podSelector)
	if err != nil {
		return "", errors.Wrap(err, "getting pod for integration test")
	}

	testCommand := runCtx.Cfg.IntegrationTest.TestCommand

	if len(testCommand) == 0 {
		return "", errors.Errorf("no TestCommand defined")
	}

	output, err := executeIntegrationTest(ctx, kubeCtl, pod, testCommand)
	return output, err
}

func executeIntegrationTest(ctx context.Context, k *kubectl.CLI, podName string, testCommand string) (string, error) {
	arguments := []string{podName, "--"}
	arguments = append(arguments, strings.Fields(testCommand)...)
	var buf bytes.Buffer

	err := k.Run(ctx, nil, &buf, "exec", arguments...)

	return buf.String(), err
}

func getPod(client kubernetes.Interface, selector string) (string, error) {
	pods, err := client.CoreV1().Pods("").List(metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return "", err
	}

	return pods.Items[0].Name, err
}
