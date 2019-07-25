package integrationtest

import (
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	kubernetesutil "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	runcontext "github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
)

func IntegrationTest(ctx context.Context, defaultLabeller *deploy.DefaultLabeller, runCtx *runcontext.RunContext) error {
	client, err := kubernetesutil.GetClientset()
	if err != nil {
		return err
	}

	kubeCtl := &kubectl.CLI{
		Namespace:   runCtx.Opts.Namespace,
		KubeContext: runCtx.KubeContext,
	}

	pod, err := getPod(client, runCtx.Opts.Namespace, runCtx.Cfg.IntegrationTest.PodSelector)
	if err != nil {
		return err
	}

	status, err := executeIntegrationTest(ctx, kubeCtl, pod, runCtx.Cfg.IntegrationTest.TestCommand)
	fmt.Printf("%+v\n", status)
	return err
}

func executeIntegrationTest(ctx context.Context, k *kubectl.CLI, podName string, testCommand string) (string, error) {
	arguments := []string{podName, "--"}
	for _, t := range strings.Fields(testCommand) {
		arguments = append(arguments, t)
	}
	b, err := k.RunOut(ctx, nil, "exec", arguments)
	return string(b), err
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
