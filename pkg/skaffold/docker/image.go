/*
Copyright 2018 Google LLC

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

package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/google/go-containerregistry/authn"
	"github.com/google/go-containerregistry/name"
	"github.com/google/go-containerregistry/v1/remote"
	"github.com/google/go-containerregistry/v1/remote/transport"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/progress"
	"github.com/moby/moby/pkg/jsonmessage"
	"github.com/moby/moby/pkg/streamformatter"
	"github.com/moby/moby/pkg/term"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BuildOptions struct {
	ImageName   string
	Dockerfile  string
	ContextDir  string
	ProgressBuf io.Writer
	BuildBuf    io.Writer
	BuildArgs   map[string]*string
}

// RunBuild performs a docker build and returns nothing
func RunBuild(ctx context.Context, cli DockerAPIClient, opts *BuildOptions) error {
	logrus.Debugf("Running docker build: context: %s, dockerfile: %s", opts.ContextDir, opts.Dockerfile)
	authConfigs, err := DefaultAuthHelper.GetAllAuthConfigs()
	if err != nil {
		return errors.Wrap(err, "read auth configs")
	}

	imageBuildOpts := types.ImageBuildOptions{
		Tags:        []string{opts.ImageName},
		Dockerfile:  opts.Dockerfile,
		BuildArgs:   opts.BuildArgs,
		AuthConfigs: authConfigs,
	}

	buildCtx, buildCtxWriter := io.Pipe()
	go func() {
		err := CreateDockerTarContext(buildCtxWriter, opts.Dockerfile, opts.ContextDir)
		if err != nil {
			buildCtxWriter.CloseWithError(errors.Wrap(err, "creating docker context"))
			return
		}
		buildCtxWriter.Close()
	}()

	progressOutput := streamformatter.NewProgressOutput(opts.ProgressBuf)
	body := progress.NewProgressReader(buildCtx, progressOutput, 0, "", "Sending build context to Docker daemon")

	resp, err := cli.ImageBuild(ctx, body, imageBuildOpts)
	if err != nil {
		return errors.Wrap(err, "docker build")
	}
	defer resp.Body.Close()
	return streamDockerMessages(opts.BuildBuf, resp.Body)
}

// TODO(@r2d4): Make this output much better, this is the bare minimum
func streamDockerMessages(dst io.Writer, src io.Reader) error {
	fd, _ := term.GetFdInfo(dst)
	return jsonmessage.DisplayJSONMessagesStream(src, dst, fd, false, nil)
}

func RunPush(ctx context.Context, cli DockerAPIClient, ref string, out io.Writer) error {
	registryAuth, err := encodedRegistryAuth(ctx, cli, DefaultAuthHelper, ref)
	if err != nil {
		return errors.Wrapf(err, "getting auth config for %s", ref)
	}
	rc, err := cli.ImagePush(ctx, ref, types.ImagePushOptions{
		RegistryAuth: registryAuth,
	})
	if err != nil {
		return errors.Wrap(err, "pushing image to repository")
	}
	defer rc.Close()
	return streamDockerMessages(out, rc)
}

func AddTag(src, target string) error {
	srcRef, err := name.ParseReference(src, name.WeakValidation)
	if err != nil {
		return errors.Wrap(err, "getting source reference")
	}

	auth, err := authn.DefaultKeychain.Resolve(srcRef.Context().Registry)
	if err != nil {
		return err
	}

	targetRef, err := name.ParseReference(target, name.WeakValidation)
	if err != nil {
		return errors.Wrap(err, "getting target reference")
	}

	return addTag(srcRef, targetRef, auth, http.DefaultTransport)
}

func addTag(ref name.Reference, targetRef name.Reference, auth authn.Authenticator, t http.RoundTripper) error {
	tr, err := transport.New(ref, auth, t, transport.PushScope)
	if err != nil {
		return err
	}

	img, err := remote.Image(ref, auth, tr)
	if err != nil {
		return err
	}

	data, err := img.RawManifest()
	if err != nil {
		return errors.Wrap(err, "getting raw manifest")
	}

	c := &http.Client{Transport: tr}
	u := url.URL{
		Scheme: transport.Scheme(ref.Context().Registry),
		Host:   targetRef.Context().RegistryStr(),
		Path:   fmt.Sprintf("/v2/%s/manifests/%s", targetRef.Context().RepositoryStr(), targetRef.Identifier()),
	}

	req, err := http.NewRequest(http.MethodPut, u.String(), bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "generating http request")
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		return nil
	default:
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("unrecognized status code during PUT: %v; %v", resp.Status, string(b))
	}
}

// Digest returns the image digest for a corresponding reference.
// The digest is of the form
// sha256:<image_id>
func Digest(ctx context.Context, cli DockerAPIClient, ref string) (string, error) {
	args := filters.KeyValuePair{Key: "reference", Value: ref}
	filters := filters.NewArgs(args)
	imageList, err := cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters,
	})
	if err != nil {
		return "", errors.Wrap(err, "getting image id")
	}
	for _, image := range imageList {
		for _, tag := range image.RepoTags {
			if tag == ref {
				return image.ID, nil
			}
		}
	}
	return "", nil
}

func RemoteDigest(identifier string) (string, error) {
	ref, err := name.ParseReference(identifier, name.WeakValidation)
	if err != nil {
		return "", errors.Wrap(err, "parsing initial ref")
	}

	auth, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return "", errors.Wrap(err, "getting default keychain auth")
	}

	img, err := remote.Image(ref, auth, http.DefaultTransport)
	if err != nil {
		return "", errors.Wrap(err, "getting image")
	}

	h, err := img.Digest()
	if err != nil {
		return "", errors.Wrap(err, "getting digest")
	}

	return h.String(), nil
}
