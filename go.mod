module github.com/GoogleContainerTools/skaffold

go 1.15

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/tektoncd/pipeline => github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8

	// pin yamlv3 to parent of https://github.com/go-yaml/yaml/commit/ae27a744346343ea814bd6f3bdd41d8669b172d0
	// Avoid indenting sequences.
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

require (
	4d63.com/tz v1.2.0
	cloud.google.com/go v0.84.0
	cloud.google.com/go/storage v1.10.0
	github.com/AlecAivazis/survey/v2 v2.2.14
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.20.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.20.0
	github.com/aws/aws-sdk-go v1.36.30 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.2.4
	github.com/buildpacks/imgutil v0.0.0-20210209163614-30601e371ce3
	github.com/buildpacks/lifecycle v0.10.2
	github.com/buildpacks/pack v0.18.1
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/docker/cli v20.10.0-beta1.0.20201117192004-5cc239616494+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.0-beta1.0.20201110211921-af34b94a78a1+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.0.0
	github.com/go-openapi/strfmt v0.19.5 // indirect
	github.com/go-openapi/validate v0.19.8 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.6
	github.com/google/go-containerregistry v0.5.1
	github.com/google/go-containerregistry/pkg/authn/k8schain v0.0.0-20210216200643-d81088d9983e // indirect
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/ko v0.8.4-0.20210615195035-ee2353837872
	github.com/google/uuid v1.1.2
	github.com/grpc-ecosystem/grpc-gateway v1.14.8
	github.com/heroku/color v0.0.6
	github.com/imdario/mergo v0.3.9
	github.com/karrick/godirwalk v1.16.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/krishicks/yaml-patch v0.0.10
	github.com/markbates/pkger v0.17.1 // indirect
	github.com/mattn/go-colorable v0.1.8
	github.com/mitchellh/go-homedir v1.1.0
	// github.com/moby/buildkit v0.7.1
	github.com/moby/buildkit v0.8.0
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.15.0 // indirect
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pkg/errors v0.9.1
	github.com/qri-io/starlib v0.4.2-0.20200213133954-ff2e8cd5ef8d // indirect
	github.com/rakyll/statik v0.1.7
	github.com/rjeczalik/notify v0.9.3-0.20201210012515-e2a77dcc14cf
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/segmentio/textio v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8
	github.com/xeipuuv/gojsonschema v1.2.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/sdk/metric v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83 // indirect
	golang.org/x/mod v0.4.2
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210603125802-9665404d3644
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56
	google.golang.org/api v0.48.0
	google.golang.org/genproto v0.0.0-20210608205507-b6d2f5bf0d7d
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.26.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	honnef.co/go/tools v0.1.3 // indirect
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/kubectl v0.19.4
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	knative.dev/pkg v0.0.0-20201119170152-e5e30edc364a // indirect
	sigs.k8s.io/kustomize/kyaml v0.11.0
	sigs.k8s.io/yaml v1.2.0
)
