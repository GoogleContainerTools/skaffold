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

	pod, err := getPod(client, runCtx.Opts.Namespace, runCtx.Cfg.IntegrationTest.PodSelector)
	if err != nil {
		return "", errors.Wrap(err, "getting pod for integration test")
	}

	output, err := executeIntegrationTest(ctx, kubeCtl, pod, runCtx.Cfg.IntegrationTest.TestCommand)
	return output, err
}

func executeIntegrationTest(ctx context.Context, k *kubectl.CLI, podName string, testCommand string) (string, error) {
	arguments := []string{podName, "--"}
	arguments = append(arguments, strings.Fields(testCommand)...)
	var buf bytes.Buffer

	err := k.Run(ctx, nil, &buf, "exec", arguments...)

	//b, err := k.RunOut(ctx, "exec", arguments...)
	return buf.String(), err
}

func getPod(client kubernetes.Interface, ns string, selector string) (string, error) {
	pods, err := client.CoreV1().Pods(ns).List(metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return "", err
	}

	return pods.Items[0].Name, err
}
