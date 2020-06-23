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

package cluster

import (
	"io/ioutil"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	pkgkubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestCreateSecret(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Touch("secret.json")
		fakeKubernetesclient := fake.NewSimpleClientset()
		t.Override(&pkgkubernetes.Client, func() (kubernetes.Interface, error) {
			return fakeKubernetesclient, nil
		})

		builder, err := NewBuilder(&runcontext.RunContext{
			Cfg: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{
							Timeout:        "20m",
							PullSecretName: "kaniko-secret",
							PullSecretPath: tmpDir.Path("secret.json"),
							Namespace:      "ns",
						},
					},
				},
			},
		})
		t.CheckNoError(err)

		// Should create a secret
		cleanup, err := builder.setupPullSecret(ioutil.Discard)
		t.CheckNoError(err)

		// Check that the secret was created
		secret, err := fakeKubernetesclient.CoreV1().Secrets("ns").Get("kaniko-secret", metav1.GetOptions{})
		t.CheckNoError(err)
		t.CheckDeepEqual("kaniko-secret", secret.GetName())
		t.CheckDeepEqual("skaffold-kaniko", secret.GetLabels()["skaffold-kaniko"])

		// Check that the secret can be deleted
		cleanup()
		_, err = fakeKubernetesclient.CoreV1().Secrets("ns").Get("kaniko-secret", metav1.GetOptions{})
		t.CheckError(true, err)
	})
}

func TestExistingSecretNotFound(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&pkgkubernetes.Client, func() (kubernetes.Interface, error) {
			return fake.NewSimpleClientset(), nil
		})

		builder, err := NewBuilder(&runcontext.RunContext{
			Cfg: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{
							Timeout:        "20m",
							PullSecretName: "kaniko-secret",
						},
					},
				},
			},
		})
		t.CheckNoError(err)

		// should fail to retrieve an existing secret
		_, err = builder.setupPullSecret(ioutil.Discard)

		t.CheckErrorContains("secret kaniko-secret does not exist. No path specified to create it", err)
	})
}

func TestExistingSecret(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&pkgkubernetes.Client, func() (kubernetes.Interface, error) {
			return fake.NewSimpleClientset(&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kaniko-secret",
				},
			}), nil
		})

		builder, err := NewBuilder(&runcontext.RunContext{
			Cfg: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{
							Timeout:        "20m",
							PullSecretName: "kaniko-secret",
						},
					},
				},
			},
		})
		t.CheckNoError(err)

		// should retrieve an existing secret
		cleanup, err := builder.setupPullSecret(ioutil.Discard)
		cleanup()

		t.CheckNoError(err)
	})
}

func TestSkipSecretCreation(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		t.Override(&pkgkubernetes.Client, func() (kubernetes.Interface, error) {
			return nil, nil
		})

		builder, err := NewBuilder(&runcontext.RunContext{
			Cfg: latest.Pipeline{
				Build: latest.BuildConfig{
					BuildType: latest.BuildType{
						Cluster: &latest.ClusterDetails{
							Timeout: "20m",
						},
					},
				},
			},
		})
		t.CheckNoError(err)

		// should retrieve an existing secret
		cleanup, err := builder.setupPullSecret(ioutil.Discard)
		cleanup()

		t.CheckNoError(err)
	})
}
