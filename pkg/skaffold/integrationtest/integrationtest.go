package integrationtest

import (
	"bytes"
	"context"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
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
	for _, t := range strings.Fields(testCommand) {
		arguments = append(arguments, t)
	}
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
