package build

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/GoogleContainerTools/kaniko/pkg/snapshot"
	kanikoutil "github.com/GoogleContainerTools/kaniko/pkg/util"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type ArtifactHash map[*latest.Artifact]string


const (
	file = "~/.skaffold/artifacts"
)

func StoreArtifacts(ctx context.Context, artifacts []*latest.Artifact) error {
	var hashes []ArtifactHash
	for _, a := range artifacts {
		key, err := getHash(ctx, a)
		if err != nil {
			
			continue
		}
		hashes[a] = key
	}
	return saveToFile(hashes) 
}

func saveToFile(hashes []ArtifactHash) error {
	os.Remove(file)
	f, err := os.Create(file)
	if err != nil {
		return errors.Wrapf(err, "creating %s", file)
	}
	defer f.Close()
	data, err := yaml.Marshal(hashes)
	if err != nil {
		return errors.Wrap(err, "marshalling hashes")
	}
	_, err = io.Copy(f, bytes.NewReader(data))
	return err
}

func ArtifactsStored(ctx context.Context, artifacts []*latest.Artifact) (bool, error) {
	stored, err := retrieveArtifacts()
	if err != nil {
		return errors.Wrap(err, "retrieving old artifacts")
	}
	for _, a := range artifacts {
		if _, ok := 
	}

}

func getHash(ctx context.Context, a *latest.Artifact) (string, error) {
	deps, err := DependenciesForArtifact(ctx, a)
	if err != nil {
		return errors.Wrapf(err, "getting dependencies for %s", a.ImageName)
	}
	lm := snapshot.NewLayeredMap(kanikoutil.Hasher(), kanikoutil.CacheHasher())
	for _, d := range deps {
		if err := lm.Add(d); err != nil {
			logrus.Warnf("Error adding %s: %v", d, err)
		}
	}
	return lm.Key()
}

func retrieveArtifacts() ([]ArtifactHash, error) {
	contents, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", file)
	}
	var hashes []ArtifactHash
	err = yaml.Unmarshal(contents, &hashes)
	return hashes, err
}
