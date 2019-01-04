package kubernetes

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/kubectl/cmd"
	"k8s.io/kubernetes/pkg/kubectl/scheme"
)

func Exec(opts cmd.StreamOptions, command []string) error {
	config, err := getClientConfig()
	if err != nil {
		return errors.Wrap(err, "getting rest config")
	}
	setConfigDefaults(config)

	client, err := internalclientset.NewForConfig(config)
	execOpts := cmd.ExecOptions{
		Config:        config,
		PodClient:     client.Core(),
		Executor:      &cmd.DefaultRemoteExecutor{},
		Command:       command,
		StreamOptions: opts,
	}

	return execOpts.Run()
}

//see https://github.com/kubernetes/client-go/blob/master/kubernetes/typed/core/v1/core_client.go#L145
func setConfigDefaults(config *rest.Config) *rest.Config {
	config.APIPath = "/api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	return config
}
