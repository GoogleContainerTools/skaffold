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

package v1beta9

import (
	"testing"

	next "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1beta10"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yaml"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpgrade(t *testing.T) {
	yaml := `apiVersion: skaffold/v1beta9
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
  - image: gcr.io/k8s-skaffold/bazel
    bazel:
      target: //mytarget
  googleCloudBuild:
    projectId: test-project
test:
  - image: gcr.io/k8s-skaffold/skaffold-example
    structureTests:
     - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s-*
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        kaniko:
          buildContext:
            gcsBucket: skaffold-kaniko
          cache: {}
      cluster:
        pullSecretName: e2esecret
        namespace: default
    test:
     - image: gcr.io/k8s-skaffold/skaffold-example
       structureTests:
         - ./test/*
    deploy:
      kubectl:
        manifests:
        - k8s-*
  - name: test local
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        docker:
          dockerfile: path/to/Dockerfile
      local:
        push: false
    deploy:
      kubectl:
        manifests:
        - k8s-*
`
	expected := `apiVersion: skaffold/v1beta10
kind: Config
build:
  artifacts:
  - image: gcr.io/k8s-skaffold/skaffold-example
    docker:
      dockerfile: path/to/Dockerfile
  - image: gcr.io/k8s-skaffold/bazel
    bazel:
      target: //mytarget
  googleCloudBuild:
    projectId: test-project
test:
  - image: gcr.io/k8s-skaffold/skaffold-example
    structureTests:
     - ./test/*
deploy:
  kubectl:
    manifests:
    - k8s-*
profiles:
  - name: test profile
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        kaniko:
          buildContext:
            gcsBucket: skaffold-kaniko
          cache: {}
      cluster:
        pullSecretName: e2esecret
        namespace: default
    test:
     - image: gcr.io/k8s-skaffold/skaffold-example
       structureTests:
         - ./test/*
    deploy:
      kubectl:
        manifests:
        - k8s-*
  - name: test local
    build:
      artifacts:
      - image: gcr.io/k8s-skaffold/skaffold-example
        docker:
          dockerfile: path/to/Dockerfile
      local:
        push: false
    deploy:
      kubectl:
        manifests:
        - k8s-*
`
	verifyUpgrade(t, yaml, expected)
}

func TestUpgradeSync(t *testing.T) {
	yaml := `
apiVersion: skaffold/v1beta9
kind: Config
build:
  artifacts:
  - image: gcr.io/no-star-1
    sync:
      '/public/A/B/a.html': /public/A/B
  - image: gcr.io/no-star-2
    sync:
      '/b.html': /www
  - image: gcr.io/no-star-3
    sync:
      'c.html': /www
  - image: gcr.io/no-star-4
    sync:
      'public/A/d.html': public/A/
  - image: gcr.io/single-star-1
    sync:
      'public/*': /app/public/
  - image: gcr.io/single-star-2
    sync:
      'main*.js': /app
  - image: gcr.io/single-star-3
    sync:
      '/public/b/*.js': /app
  - image: gcr.io/single-star-4
    sync:
      '/c/prefix-*': /app
  - image: gcr.io/k8s-skaffold/node-example
    sync:
      '**/*.js': .
  - image: gcr.io/k8s-skaffold/react-reload
    sync:
      'src/***/*.js': app/
  - image: nginx
profiles:
- name: test-profile-migration
  build:
    artifacts:
    - image: gcr.io/k8s-skaffold/node-example
      sync:
        '**/*.js': .
deploy:
  kubectl:
    manifests:
    - "backend/k8s/**"
`
	expected := `
apiVersion: skaffold/v1beta10
kind: Config
build:
  artifacts:
  - image: gcr.io/no-star-1
    sync:
      manual:
      - src: /public/A/B/a.html
        dest: /
  - image: gcr.io/no-star-2
    sync:
      manual:
      - src: /b.html
        dest: /www
        strip: /
  - image: gcr.io/no-star-3
    sync:
      manual:
      - src: c.html
        dest: /www
  - image: gcr.io/no-star-4
    sync:
      manual:
      - src: public/A/d.html
        dest: .
  - image: gcr.io/single-star-1
    sync:
      manual:
      - src: 'public/*'
        dest: /app/
  - image: gcr.io/single-star-2
    sync:
      manual:
      - src: 'main*.js'
        dest: /app
  - image: gcr.io/single-star-3
    sync:
      manual:
      - src: '/public/b/*.js'
        dest: /app
        strip: /public/b/
  - image: gcr.io/single-star-4
    sync:
      manual:
      - src: '/c/prefix-*'
        dest: /app
        strip: /c/
  - image: gcr.io/k8s-skaffold/node-example
    sync:
      manual:
      - src: '**/*.js'
        dest: .
  - image: gcr.io/k8s-skaffold/react-reload
    sync:
      manual:
      - src: 'src/**/*.js'
        dest: app/
        strip: src/
  - image: nginx
profiles:
- name: test-profile-migration
  build:
    artifacts:
    - image: gcr.io/k8s-skaffold/node-example
      sync:
        manual:
        - src: '**/*.js'
          dest: .
deploy:
  kubectl:
    manifests:
    - "backend/k8s/**"
`
	verifyUpgrade(t, yaml, expected)
}

func verifyUpgrade(t *testing.T, input, output string) {
	config := NewSkaffoldConfig()
	err := yaml.UnmarshalStrict([]byte(input), config)
	testutil.CheckErrorAndDeepEqual(t, false, err, Version, config.GetVersion())

	upgraded, err := config.Upgrade()
	testutil.CheckError(t, false, err)

	expected := next.NewSkaffoldConfig()
	err = yaml.UnmarshalStrict([]byte(output), expected)

	testutil.CheckErrorAndDeepEqual(t, false, err, expected, upgraded)
}
