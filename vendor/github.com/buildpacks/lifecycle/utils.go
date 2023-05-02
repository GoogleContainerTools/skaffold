package lifecycle

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/buildpacks/lifecycle/buildpack"
)

func WriteTOML(path string, data interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(data)
}

func ReadGroup(path string) (buildpack.Group, error) {
	var group buildpack.Group
	_, err := toml.DecodeFile(path, &group)
	return group, err
}

func ReadOrder(path string) (buildpack.Order, error) {
	var order struct {
		Order buildpack.Order `toml:"order"`
	}
	_, err := toml.DecodeFile(path, &order)
	return order.Order, err
}

func TruncateSha(sha string) string {
	rawSha := strings.TrimPrefix(sha, "sha256:")
	if len(sha) > 12 {
		return rawSha[0:12]
	}
	return rawSha
}

func removeStagePrefixes(mixins []string) []string {
	var result []string
	for _, m := range mixins {
		s := strings.SplitN(m, ":", 2)
		if len(s) == 1 {
			result = append(result, s[0])
		} else {
			result = append(result, s[1])
		}
	}
	return result
}
