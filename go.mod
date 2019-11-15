module github.com/GoogleContainerTools/skaffold

go 1.12

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.0.1+incompatible
	github.com/Azure/go-autorest/autorest/adal => github.com/Azure/go-autorest/autorest/adal v0.6.1-0.20190906230412-69b4126ece6b
	github.com/Azure/go-autorest/autorest/date => github.com/Azure/go-autorest/autorest/date v0.2.1-0.20190906230412-69b4126ece6b
	github.com/Azure/go-autorest/autorest/mocks => github.com/Azure/go-autorest/autorest/mocks v0.3.1-0.20190906230412-69b4126ece6b
	github.com/Azure/go-autorest/tracing => github.com/Azure/go-autorest/tracing v0.5.0
	github.com/containerd/containerd => github.com/containerd/containerd v1.3.0
	github.com/docker/docker => github.com/docker/docker v1.4.2-0.20190924003213-a8608b5b67c7
	golang.org/x/crypto v0.0.0-20190129210102-0709b304e793 => golang.org/x/crypto v0.0.0-20180904163835-0709b304e793
	k8s.io/api => k8s.io/api v0.0.0-20190620084959-7cf5895f2711
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190612205821-1799e75a0719
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190620085101-78d2af792bab
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20190831163037-3b58a944563f
	k8s.io/kubernetes => k8s.io/kubernetes v1.12.10
)

require (
	4d63.com/embedfiles v1.0.0 // indirect
	4d63.com/tz v1.1.0
	cloud.google.com/go v0.47.0 // indirect
	cloud.google.com/go/bigquery v1.2.0 // indirect
	cloud.google.com/go/storage v1.3.0
	contrib.go.opencensus.io/exporter/prometheus v0.1.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.8 // indirect
	github.com/Azure/azure-sdk-for-go v36.1.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.9.2 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667 // indirect
	github.com/aws/aws-sdk-go v1.25.33 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.1.5
	github.com/creack/pty v1.1.9 // indirect
	github.com/docker/cli v0.0.0-20191113002236-6c12a82f3306
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20191011121108-aa519ddbe484 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gobuffalo/envy v1.7.1 // indirect
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.3.1
	github.com/google/go-containerregistry v0.0.0-20191114011901-9087f290f1e4
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gophercloud/gophercloud v0.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.1
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/imdario/mergo v0.3.8
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/karrick/godirwalk v1.13.0
	github.com/krishicks/yaml-patch v0.0.10
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.6.3
	github.com/onsi/ginkgo v1.10.3 // indirect
	github.com/onsi/gomega v1.7.1 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1.0.20190228220655-ac19fd6e7483
	github.com/opencontainers/image-spec v1.0.1
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1 // indirect
	github.com/prometheus/procfs v0.0.6 // indirect
	github.com/rakyll/statik v0.1.6
	github.com/rjeczalik/notify v0.9.2
	github.com/russross/blackfriday/v2 v2.0.1
	github.com/segmentio/textio v1.2.0
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8
	go.opencensus.io v0.22.2 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.12.0 // indirect
	golang.org/x/crypto v0.0.0-20191112222119-e1110fd1c708
	golang.org/x/net v0.0.0-20191112182307-2180aed22343 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	golang.org/x/sys v0.0.0-20191110163157-d32e6e3b99c4 // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.0.0-20191112195655-aa38f8e97acc // indirect
	golang.org/x/xerrors v0.0.0-20191011141410-1b5146add898 // indirect
	google.golang.org/api v0.13.0
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191114150713-6bbd007550de
	google.golang.org/grpc v1.25.1
	gopkg.in/AlecAivazis/survey.v1 v1.8.7
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.5
	k8s.io/api v0.0.0-20190831074750-7364b6bdad65
	k8s.io/apimachinery v0.0.0-20190831074630-461753078381
	k8s.io/client-go v0.0.0-20190831074946-3fe2abece89e
	k8s.io/kubectl v0.0.0-20190831163037-3b58a944563f
	k8s.io/kubernetes v1.12.10 // indirect
	k8s.io/utils v0.0.0-20190801114015-581e00157fb1
	knative.dev/pkg v0.0.0-20190730155243-972acd413fb9 // indirect
)
