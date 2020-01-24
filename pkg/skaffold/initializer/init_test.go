package initializer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestDoInit(t *testing.T) {
	tests := []struct {
		name      string
		dir       string
		config    Config
		shouldErr bool
	}{
		//TODO: mocked kompose test
		{
			name: "getting-started",
			dir:  "testdata/init/hello",
			config: Config{
				Force: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "ignore existing tags",
			dir:  "testdata/init/ignore-tags",
			config: Config{
				Force: true,
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "microservices (backwards compatibility)",
			dir:  "testdata/init/microservices",
			config: Config{
				Force: true,
				CliArtifacts: []string{
					"leeroy-app/Dockerfile=gcr.io/k8s-skaffold/leeroy-app",
					"leeroy-web/Dockerfile=gcr.io/k8s-skaffold/leeroy-web",
				},
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "microservices",
			dir:  "testdata/init/microservices",
			config: Config{
				Force: true,
				CliArtifacts: []string{
					`{"builder":"Docker","payload":{"path":"leeroy-app/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-app"}`,
					`{"builder":"Docker","payload":{"path":"leeroy-web/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-web"}`,
				},
				Opts: config.SkaffoldOptions{
					ConfigurationFile: "skaffold.yaml.out",
				},
			},
		},
		{
			name: "error writing config file",
			dir:  "testdata/init/microservices",

			config: Config{
				Force: true,
				CliArtifacts: []string{
					`{"builder":"Docker","payload":{"path":"leeroy-app/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-app"}`,
					`{"builder":"Docker","payload":{"path":"leeroy-web/Dockerfile"},"image":"gcr.io/k8s-skaffold/leeroy-web"}`,
				},
				Opts: config.SkaffoldOptions{
					// erroneous config file as . is a directory
					ConfigurationFile: ".",
				},
			},
			shouldErr: true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Chdir(test.dir)
			err := DoInit(context.TODO(), os.Stdout, test.config)
			t.CheckError(test.shouldErr, err)
			checkGeneratedConfig(t, ".")
		})
	}
}

func TestDoInitAnalyze(t *testing.T) {
	tests := []struct {
		name        string
		dir         string
		config      Config
		expectedOut string
		shouldErr   bool
	}{
		{
			name: "analyze microservices",
			dir:  "testdata/init/microservices",
			config: Config{
				Force:   true,
				Analyze: true,
			},
			expectedOut: strip(`{
							"dockerfiles":["leeroy-app/Dockerfile","leeroy-web/Dockerfile"],
							"images":["gcr.io/k8s-skaffold/leeroy-app","gcr.io/k8s-skaffold/leeroy-web"]
							}`),
		},
		{
			name: "analyze microservices new format",
			dir:  "testdata/init/microservices",
			config: Config{
				Force:         true,
				Analyze:       true,
				EnableJibInit: true,
			},
			expectedOut: strip(`{
									"builders":[
										{"name":"Docker","payload":{"path":"leeroy-app/Dockerfile"}},
										{"name":"Docker","payload":{"path":"leeroy-web/Dockerfile"}}
									],
									"images":[
										{"name":"gcr.io/k8s-skaffold/leeroy-app","foundMatch":false},
										{"name":"gcr.io/k8s-skaffold/leeroy-web","foundMatch":false}]}`),
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			var out bytes.Buffer
			t.Chdir(test.dir)
			err := DoInit(context.TODO(), &out, test.config)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedOut, out.String())
		})
	}
}

func strip(s string) string {
	cutString := "\n\t\r"
	stripped := ""
	for _, r := range s {
		if strings.ContainsRune(cutString, r) {
			continue
		}
		stripped = fmt.Sprintf("%s%c", stripped, r)
	}
	return stripped
}

func checkGeneratedConfig(t *testutil.T, dir string) {
	expectedOutput, err := schema.ParseConfig(filepath.Join(dir, "skaffold.yaml"), false)
	t.CheckNoError(err)

	output, err := schema.ParseConfig(filepath.Join(dir, "skaffold.yaml.out"), false)
	t.CheckNoError(err)
	t.CheckDeepEqual(expectedOutput, output)
}
