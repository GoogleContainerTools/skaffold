module github.com/GoogleContainerTools/skaffold

go 1.15

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8

	// pin yamlv3 to parent of https://github.com/go-yaml/yaml/commit/ae27a744346343ea814bd6f3bdd41d8669b172d0
	// Avoid indenting sequences.
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c

	// TODO(halvards) Pin to match other k8s.io modules. Necessary since ko is now on v0.22.x.
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.3
)

// TODO(bdealwis)[10/07/2021] CVE-2021-41103: pin github.com/containerd/containerd v1.5.7
// TODO(bdealwis)[10/07/2021] CVE-2021-30465: pin github.com/opencontainers/runc v1.0.2
// TODO(bdealwis)[10/07/2021] GO-2021-0113: pin golang.org/x/text v0.3.7

require (
	4d63.com/tz v1.2.0
	cloud.google.com/go/container v1.0.0 // indirect
	cloud.google.com/go/monitoring v1.0.0
	cloud.google.com/go/profiler v0.1.0
	cloud.google.com/go/storage v1.16.0
	cloud.google.com/go/trace v1.0.0 // indirect
	github.com/AlecAivazis/survey/v2 v2.2.15
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.20.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.20.0
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/ahmetalpbalkan/dlog v0.0.0-20170105205344-4fb5f8204f26 // indirect
	github.com/ahmetb/dlog v0.0.0-20170105205344-4fb5f8204f26
	github.com/aws/aws-sdk-go v1.36.30 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.3.4
	github.com/buildpacks/imgutil v0.0.0-20210209163614-30601e371ce3
	github.com/buildpacks/lifecycle v0.10.2
	github.com/buildpacks/pack v0.18.1
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/docker/cli v20.10.7+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.10+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/dustin/go-humanize v1.0.0
	github.com/evanphx/json-patch v4.11.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.4.2
	github.com/golang/glog v0.0.0-20210429001901-424d2337a529
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.6.0
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210216200643-d81088d9983e // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/ko v0.9.4-0.20211102213605-98211906053b
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.5.0
	github.com/heroku/color v0.0.6
	github.com/imdario/mergo v0.3.12
	github.com/karrick/godirwalk v1.16.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/krishicks/yaml-patch v0.0.10
	github.com/mattn/go-colorable v0.1.8
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.8.0
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.2-0.20210730191737-8e42a01fb1b7
	github.com/otiai10/copy v1.6.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pkg/errors v0.9.1
	github.com/rakyll/statik v0.1.7
	github.com/rjeczalik/notify v0.9.3-0.20201210012515-e2a77dcc14cf
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/segmentio/textio v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8
	github.com/xeipuuv/gojsonschema v1.2.0
	go.lsp.dev/protocol v0.11.2
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/sdk/metric v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56
	golang.org/x/tools v0.1.7
	google.golang.org/api v0.57.0
	google.golang.org/genproto v0.0.0-20211005153810-c76a74d43a8e
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v0.21.3
	k8s.io/kubectl v0.21.3
	k8s.io/utils v0.0.0-20210517184530-5a248b5acedc
	knative.dev/pkg v0.0.0-20201119170152-e5e30edc364a // indirect
	sigs.k8s.io/cli-utils v0.22.0
	sigs.k8s.io/kustomize/kyaml v0.10.17
	sigs.k8s.io/yaml v1.2.0
)
