package build

import (
	"context"
	"io"
	"io/ioutil"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kaniko"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KanikoBuilder struct {
	*v1alpha2.BuildConfig
}

func NewKanikoBuilder(cfg *v1alpha2.BuildConfig) (*KanikoBuilder, error) {
	return &KanikoBuilder{
		BuildConfig: cfg,
	}, nil
}

func (k *KanikoBuilder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*v1alpha2.Artifact) (*BuildResult, error) {
	res := &BuildResult{}

	client, err := kubernetes.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "getting kubernetes client")
	}

	secretData, err := ioutil.ReadFile(k.KanikoBuild.PullSecret)
	if err != nil {
		return nil, errors.Wrap(err, "reading secret")
	}

	_, err = client.CoreV1().Secrets("default").Create(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kaniko-secret",
			Labels: map[string]string{"kaniko": "kaniko"},
		},
		Data: map[string][]byte{
			"kaniko-secret": secretData,
		},
	})
	if err != nil {
		logrus.Warnf("creating secret: %s", err)
	}
	defer func() {
		if err := client.CoreV1().Secrets("default").Delete("kaniko-secret", &metav1.DeleteOptions{}); err != nil {
			logrus.Warnf("deleting secret")
		}
	}()

	// TODO(r2d4): parallel builds
	for _, artifact := range artifacts {
		initialTag, err := kaniko.RunKanikoBuild(ctx, out, artifact, k.KanikoBuild)
		if err != nil {
			return nil, errors.Wrapf(err, "running kaniko build for %s", artifact.ImageName)
		}

		digest, err := docker.RemoteDigest(initialTag)
		if err != nil {
			return nil, errors.Wrap(err, "getting digest")
		}

		tag, err := tagger.GenerateFullyQualifiedImageName(artifact.Workspace, &tag.TagOptions{
			ImageName: artifact.ImageName,
			Digest:    digest,
		})
		if err != nil {
			return nil, errors.Wrap(err, "generating tag")
		}

		if err := docker.AddTag(initialTag, tag); err != nil {
			return nil, errors.Wrap(err, "tagging image")
		}

		res.Builds = append(res.Builds, Build{
			ImageName: artifact.ImageName,
			Tag:       tag,
			Artifact:  artifact,
		})
	}
	return res, nil
}
