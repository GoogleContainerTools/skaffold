module github.com/GoogleContainerTools/skaffold/v2

go 1.19

// gnostic was replaced with v0.4.1 in
// https://github.com/GoogleContainerTools/skaffold/pull/5038, but why..?
exclude github.com/googleapis/gnostic v0.5.1

// image-spec was replaced with an older version in
// https://github.com/GoogleContainerTools/skaffold/pull/6979, but why..?
exclude (
	github.com/opencontainers/image-spec v1.0.2
	github.com/opencontainers/image-spec v1.0.2-0.20211117181255-693428a734f5
	github.com/opencontainers/image-spec v1.0.3-0.20220114050600-8b9d41f48198
)

// pin yamlv3 to parent of https://github.com/go-yaml/yaml/commit/ae27a744346343ea814bd6f3bdd41d8669b172d0
// Avoid indenting sequences.
exclude (
	gopkg.in/yaml.v3 v3.0.0-20200605160147-a5ece683394c
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

// pin otel to avoid breaking instrumentation package, we can remove this block after switching metric collecting approach and export platform
exclude (
	go.opentelemetry.io/otel v1.13.0
	go.opentelemetry.io/otel v1.7.0
	go.opentelemetry.io/otel/sdk v1.13.0
	go.opentelemetry.io/otel/sdk v1.7.0
)

// TODO(halvards) Pin to match other k8s.io modules. Necessary since ko is on a later version.
exclude k8s.io/apimachinery v0.23.5

// TODO(bdealwis)[10/07/2021] CVE-2021-30465: pin github.com/opencontainers/runc v1.0.2
// TODO(bdealwis)[10/07/2021] GO-2021-0113: pin golang.org/x/text v0.3.7

require (
	4d63.com/tz v1.2.0
	cloud.google.com/go/monitoring v1.8.0
	cloud.google.com/go/profiler v0.1.0
	cloud.google.com/go/storage v1.27.0
	github.com/AlecAivazis/survey/v2 v2.2.15
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.20.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v0.20.0
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/ahmetb/dlog v0.0.0-20170105205344-4fb5f8204f26
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.3.4
	github.com/buildpacks/imgutil v0.0.0-20211203200417-76206845baac
	github.com/buildpacks/lifecycle v0.13.5
	github.com/buildpacks/pack v0.25.0
	github.com/cenkalti/backoff/v4 v4.1.3
	github.com/containerd/containerd v1.5.10
	github.com/docker/cli v20.10.17+incompatible
	github.com/docker/distribution v2.8.1+incompatible
	github.com/docker/docker v20.10.17+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/dustin/go-humanize v1.0.1
	github.com/fatih/semgroup v1.2.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.4.2
	github.com/golang/glog v1.0.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.9
	github.com/google/go-containerregistry v0.11.0
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/ko v0.11.2
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3
	github.com/heroku/color v0.0.6
	github.com/imdario/mergo v0.3.12
	github.com/joho/godotenv v1.3.0
	github.com/karrick/godirwalk v1.16.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/krishicks/yaml-patch v0.0.10
	github.com/mattn/go-colorable v0.1.13
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.8.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0-rc2
	github.com/otiai10/copy v1.6.0
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/pkg/errors v0.9.1
	github.com/rjeczalik/notify v0.9.3-0.20201210012515-e2a77dcc14cf
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/segmentio/textio v1.2.0
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/afero v1.8.2
	github.com/spf13/cobra v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8
	github.com/xeipuuv/gojsonschema v1.2.0
	go.lsp.dev/jsonrpc2 v0.9.0
	go.lsp.dev/protocol v0.11.2
	go.lsp.dev/uri v0.3.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/stdout v0.20.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.20.0
	go.opentelemetry.io/otel/metric v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/sdk/metric v0.20.0
	go.opentelemetry.io/otel/trace v1.7.0
	golang.org/x/oauth2 v0.4.0
	golang.org/x/sync v0.1.0
	golang.org/x/sys v0.5.0
	golang.org/x/term v0.4.0
	golang.org/x/tools v0.5.0
	google.golang.org/api v0.103.0
	google.golang.org/genproto v0.0.0-20230202175211-008b39050e57
	google.golang.org/grpc v1.53.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.23.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.23.5
	k8s.io/kubectl v0.21.6
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/cli-utils v0.22.0
	sigs.k8s.io/kustomize/kyaml v0.10.17
	sigs.k8s.io/yaml v1.3.0
)

require (
	4d63.com/embedfiles v0.0.0-20190311033909-995e0740726f // indirect
	cloud.google.com/go v0.107.0 // indirect
	cloud.google.com/go/compute v1.15.1 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	cloud.google.com/go/iam v0.8.0 // indirect
	cloud.google.com/go/trace v1.4.0 // indirect
	github.com/Azure/azure-sdk-for-go v66.0.0+incompatible // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.28 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.21 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.11 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/BurntSushi/toml v1.1.0 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.0 // indirect
	github.com/Microsoft/hcsshim v0.9.6 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/ahmetalpbalkan/dlog v0.0.0-20170105205344-4fb5f8204f26 // indirect
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/apex/log v1.9.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/aws/aws-sdk-go-v2 v1.16.16 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.17.8 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.12.21 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.13.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.16.19 // indirect
	github.com/aws/smithy-go v1.13.3 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20220228164355-396b2034c795 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chrismellard/docker-credential-acr-env v0.0.0-20220119192733-fe33c00cee21 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.12.0 // indirect
	github.com/containerd/ttrpc v1.1.0 // indirect
	github.com/containerd/typeurl v1.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/docker-credential-helpers v0.6.4 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dprotaso/go-yit v0.0.0-20191028211022-135eb7262960 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/evanphx/json-patch v4.12.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/gdamore/tcell/v2 v2.5.0 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-openapi/analysis v0.21.4 // indirect
	github.com/go-openapi/errors v0.20.3 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/loads v0.21.2 // indirect
	github.com/go-openapi/runtime v0.24.2 // indirect
	github.com/go-openapi/spec v0.20.7 // indirect
	github.com/go-openapi/strfmt v0.21.3 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-openapi/validate v0.22.0 // indirect
	github.com/gobuffalo/here v0.6.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.4.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20210804190019-f964ff605595 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.0 // indirect
	github.com/googleapis/gax-go/v2 v2.7.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/klauspost/compress v1.15.15 // indirect
	github.com/letsencrypt/boulder v0.0.0-20220929215747-76583552c2be // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/markbates/pkger v0.17.1 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/ioprogress v0.0.0-20180201004757-6a23b12fa88e // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/opencontainers/runc v1.1.4 // indirect
	github.com/opencontainers/selinux v1.11.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.13.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/rivo/tview v0.0.0-20210624165335-29d673af0ce2 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20201211074657-223ce5d391b0 // indirect
	github.com/segmentio/encoding v0.2.7 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/sigstore/cosign v1.13.1 // indirect
	github.com/sigstore/rekor v0.12.1-0.20220915152154-4bb6f441c1b2 // indirect
	github.com/sigstore/sigstore v1.4.4 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.13.0 // indirect
	github.com/src-d/gcfg v1.4.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/subosito/gotenv v1.4.1 // indirect
	github.com/theupdateframework/go-tuf v0.5.2 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/vbatts/tar-split v0.11.2 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca // indirect
	go.lsp.dev/pkg v0.0.0-20210323044036-f7deec69b52e // indirect
	go.mongodb.org/mongo-driver v1.10.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.20.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/automaxprocs v1.4.1-0.20220314153950-975e177ad84f // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/crypto v0.5.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/time v0.0.0-20220922220347-f3bd1da661af // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/src-d/go-billy.v4 v4.3.2 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/klog/v2 v2.60.1-0.20220317184644-43cc75f9ae89 // indirect
	k8s.io/kube-openapi v0.0.0-20220124234850-424119656bbf // indirect
	knative.dev/pkg v0.0.0-20211004133827-74ac82a333a4 // indirect
	sigs.k8s.io/kind v0.12.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
)
