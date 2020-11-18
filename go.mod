module github.com/GoogleContainerTools/skaffold

go 1.15

replace (
	github.com/containerd/containerd => github.com/containerd/containerd v1.3.4
	github.com/docker/docker => github.com/docker/docker v1.4.2-0.20200221181110-62bd5a33f707
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8

	// pin yamlv3 to parent of https://github.com/go-yaml/yaml/commit/ae27a744346343ea814bd6f3bdd41d8669b172d0
	// Avoid indenting sequences.
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c

	k8s.io/api => k8s.io/api v0.17.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.4
	k8s.io/client-go => k8s.io/client-go v0.17.4
	k8s.io/kubectl => k8s.io/kubectl v0.17.4
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.10
)

require (
	4d63.com/tz v1.1.0
	cloud.google.com/go/storage v1.8.0
	contrib.go.opencensus.io/exporter/ocagent v0.6.0 // indirect
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.13.1 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.2.4
	github.com/buildpacks/imgutil v0.0.0-20201022190551-6525b8cdcdd0
	github.com/buildpacks/lifecycle v0.9.3
	github.com/buildpacks/pack v0.15.0
	github.com/cenkalti/backoff/v4 v4.0.2
	github.com/docker/cli v0.0.0-20200312141509-ef2f64abbd37
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c
	github.com/docker/go-connections v0.4.0
	github.com/dustin/go-humanize v1.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.0.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/golang/protobuf v1.4.3
	github.com/google/go-cmp v0.5.2
	github.com/google/go-containerregistry v0.1.4
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/martian v2.1.1-0.20190517191504-25dcb96d9e51+incompatible // indirect
	github.com/google/uuid v1.1.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.3
	github.com/heroku/color v0.0.6
	github.com/imdario/mergo v0.3.9
	github.com/karrick/godirwalk v1.15.6
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/krishicks/yaml-patch v0.0.10
	github.com/mattn/go-colorable v0.1.8
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.7.1
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/openzipkin/zipkin-go v0.2.2 // indirect
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/rakyll/statik v0.1.7
	github.com/rjeczalik/notify v0.9.2
	github.com/russross/blackfriday/v2 v2.0.1
	github.com/segmentio/textio v1.2.0
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8
	github.com/xeipuuv/gojsonschema v1.2.0
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.12.0 // indirect
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/mod v0.3.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/api v0.25.0
	google.golang.org/genproto v0.0.0-20201022181438-0ff5f38871d5
	google.golang.org/grpc v1.33.1
	gopkg.in/AlecAivazis/survey.v1 v1.8.8
	gopkg.in/yaml.v2 v2.3.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.18.8
	k8s.io/apiextensions-apiserver v0.18.1 // indirect
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.18.8
	k8s.io/kubectl v0.0.0-20190831163037-3b58a944563f
	k8s.io/utils v0.0.0-20200324210504-a9aa75ae1b89
	knative.dev/pkg v0.0.0-20200416021448-f68639f04b39 // indirect
	sigs.k8s.io/kustomize/kyaml v0.9.2
	sigs.k8s.io/yaml v1.2.0
)
