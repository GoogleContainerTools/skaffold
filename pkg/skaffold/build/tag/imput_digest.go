package tag

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sort"
)

type inputDigestTagger struct {
	getDependencies func(ctx context.Context, artifact *latest.Artifact) ([]string, error)
}

func NewInputDigestTagger(getDependencies func(ctx context.Context, artifact *latest.Artifact) ([]string, error)) (Tagger, error) {
	return &inputDigestTagger{
		getDependencies: getDependencies,
	}, nil
}

func (t *inputDigestTagger) GenerateTag(_ string, image *latest.Artifact) (string, error) {
	ctx := context.Background()
	var inputs []string

	dependencies, err := t.getDependencies(ctx, image)
	if err != nil {
		return "", err
	}

	sort.Strings(dependencies)

	for _, d := range dependencies {
		h, err := fileHasher(d)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Tracef("skipping dependency for artifact cache calculation, file not found %s: %s", d, err)
				continue // Ignore files that don't exist
			}

			return "", fmt.Errorf("getting hash for %q: %w", d, err)
		}
		inputs = append(inputs, h)
	}

	return encode(inputs)
}

func encode(inputs []string) (string, error) {
	// get a key for the hashes
	hasher := sha256.New()
	enc := json.NewEncoder(hasher)
	if err := enc.Encode(inputs); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// fileHasher hashes the contents and name of a file
func fileHasher(p string) (string, error) {
	h := md5.New()
	fi, err := os.Lstat(p)
	if err != nil {
		return "", err
	}
	h.Write([]byte(fi.Mode().String()))
	h.Write([]byte(fi.Name()))
	if fi.Mode().IsRegular() {
		f, err := os.Open(p)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
