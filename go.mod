module github.com/GoogleContainerTools/skaffold/v2

go 1.23

// these require code change may remove these later
exclude (
	github.com/opencontainers/image-spec v1.1.0-rc3
	github.com/opencontainers/image-spec v1.1.0-rc4
)

// Unit tests fail due to a breaking change in reference.Parse() from this version.
exclude github.com/docker/distribution v2.8.3+incompatible

// this version requires code change may remove these later
exclude go.opentelemetry.io/otel/metric v0.37.0

// doesn't work well with windows
exclude github.com/karrick/godirwalk v1.17.0

require (
	4d63.com/tz v1.2.0
	cloud.google.com/go/cloudbuild v1.19.0
	cloud.google.com/go/monitoring v1.22.0
	cloud.google.com/go/profiler v0.4.1
	cloud.google.com/go/storage v1.47.0
	github.com/AlecAivazis/survey/v2 v2.2.15
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.49.0
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace v1.21.0
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d
	github.com/ahmetb/dlog v0.0.0-20170105205344-4fb5f8204f26
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.3.4
	github.com/buildpacks/imgutil v0.0.0-20240605145725-186f89b2d168
	github.com/buildpacks/lifecycle v0.20.4
	github.com/buildpacks/pack v0.35.1
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/containerd/containerd v1.7.23
	github.com/distribution/reference v0.6.0
	github.com/docker/cli v27.3.1+incompatible
	github.com/docker/distribution v2.8.2+incompatible
	github.com/docker/docker v27.3.1+incompatible
	github.com/docker/go-connections v0.5.0
	github.com/dustin/go-humanize v1.0.1
	github.com/evanphx/json-patch v4.12.0+incompatible
	github.com/fatih/semgroup v1.2.0
	github.com/go-git/go-git/v5 v5.12.0
	github.com/golang/glog v1.2.2
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8
	github.com/golang/protobuf v1.5.4
	github.com/google/go-cmp v0.6.0
	github.com/google/go-containerregistry v0.20.2
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/ko v0.14.0
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.14.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.23.0
	github.com/heroku/color v0.0.6
	github.com/imdario/mergo v0.3.16
	github.com/joho/godotenv v1.4.0
	github.com/karrick/godirwalk v1.16.1
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/krishicks/yaml-patch v0.0.10
	github.com/letsencrypt/boulder v0.0.0-20231026200631-000cd05d5491
	github.com/mattn/go-colorable v0.1.13
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.17.1
	github.com/moby/patternmatcher v0.6.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.1.0
	github.com/otiai10/copy v1.14.0
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/pkg/errors v0.9.1
	github.com/rjeczalik/notify v0.9.3
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/segmentio/encoding v0.2.7
	github.com/segmentio/textio v1.2.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/afero v1.11.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399
	github.com/xeipuuv/gojsonschema v1.2.0
	go.lsp.dev/jsonrpc2 v0.9.0
	go.lsp.dev/protocol v0.11.2
	go.lsp.dev/uri v0.3.0
	go.opentelemetry.io/otel v1.32.0
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.32.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.32.0
	go.opentelemetry.io/otel/metric v1.32.0
	go.opentelemetry.io/otel/sdk v1.32.0
	go.opentelemetry.io/otel/sdk/metric v1.32.0
	go.opentelemetry.io/otel/trace v1.32.0
	golang.org/x/crypto v0.30.0
	golang.org/x/oauth2 v0.24.0
	golang.org/x/sync v0.10.0
	golang.org/x/sys v0.28.0
	golang.org/x/term v0.27.0
	golang.org/x/tools v0.28.0
	google.golang.org/api v0.210.0
	google.golang.org/genproto v0.0.0-20241202173237-19429a94021a
	google.golang.org/genproto/googleapis/api v0.0.0-20241202173237-19429a94021a
	google.golang.org/grpc v1.68.1
	google.golang.org/protobuf v1.35.2
	gopkg.in/go-jose/go-jose.v2 v2.6.3
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.28.3
	k8s.io/apimachinery v0.28.3
	k8s.io/client-go v0.28.3
	k8s.io/kubectl v0.21.6
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b
	sigs.k8s.io/cli-utils v0.22.0
	sigs.k8s.io/kustomize/api v0.8.8
	sigs.k8s.io/kustomize/kyaml v0.10.17
	sigs.k8s.io/yaml v1.4.0
)

require (
	4d63.com/embedfiles v0.0.0-20190311033909-995e0740726f // indirect
	cel.dev/expr v0.19.0 // indirect
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/accessapproval v1.8.2 // indirect
	cloud.google.com/go/accesscontextmanager v1.9.2 // indirect
	cloud.google.com/go/aiplatform v1.69.0 // indirect
	cloud.google.com/go/analytics v0.25.2 // indirect
	cloud.google.com/go/apigateway v1.7.2 // indirect
	cloud.google.com/go/apigeeconnect v1.7.2 // indirect
	cloud.google.com/go/apigeeregistry v0.9.2 // indirect
	cloud.google.com/go/apikeys v1.2.2 // indirect
	cloud.google.com/go/appengine v1.9.2 // indirect
	cloud.google.com/go/area120 v0.9.2 // indirect
	cloud.google.com/go/artifactregistry v1.16.0 // indirect
	cloud.google.com/go/asset v1.20.3 // indirect
	cloud.google.com/go/assuredworkloads v1.12.2 // indirect
	cloud.google.com/go/auth v0.12.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.6 // indirect
	cloud.google.com/go/automl v1.14.2 // indirect
	cloud.google.com/go/baremetalsolution v1.3.2 // indirect
	cloud.google.com/go/batch v1.11.3 // indirect
	cloud.google.com/go/beyondcorp v1.1.2 // indirect
	cloud.google.com/go/bigquery v1.64.0 // indirect
	cloud.google.com/go/bigtable v1.33.0 // indirect
	cloud.google.com/go/billing v1.20.0 // indirect
	cloud.google.com/go/binaryauthorization v1.9.2 // indirect
	cloud.google.com/go/certificatemanager v1.9.2 // indirect
	cloud.google.com/go/channel v1.19.1 // indirect
	cloud.google.com/go/clouddms v1.8.2 // indirect
	cloud.google.com/go/cloudtasks v1.13.2 // indirect
	cloud.google.com/go/compute v1.29.0 // indirect
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	cloud.google.com/go/contactcenterinsights v1.16.0 // indirect
	cloud.google.com/go/container v1.42.0 // indirect
	cloud.google.com/go/containeranalysis v0.13.2 // indirect
	cloud.google.com/go/datacatalog v1.24.0 // indirect
	cloud.google.com/go/dataflow v0.10.2 // indirect
	cloud.google.com/go/dataform v0.10.2 // indirect
	cloud.google.com/go/datafusion v1.8.2 // indirect
	cloud.google.com/go/datalabeling v0.9.2 // indirect
	cloud.google.com/go/dataplex v1.20.0 // indirect
	cloud.google.com/go/dataproc v1.12.0 // indirect
	cloud.google.com/go/dataproc/v2 v2.10.0 // indirect
	cloud.google.com/go/dataqna v0.9.2 // indirect
	cloud.google.com/go/datastore v1.20.0 // indirect
	cloud.google.com/go/datastream v1.12.0 // indirect
	cloud.google.com/go/deploy v1.26.0 // indirect
	cloud.google.com/go/dialogflow v1.62.0 // indirect
	cloud.google.com/go/dlp v1.20.0 // indirect
	cloud.google.com/go/documentai v1.35.0 // indirect
	cloud.google.com/go/domains v0.10.2 // indirect
	cloud.google.com/go/edgecontainer v1.4.0 // indirect
	cloud.google.com/go/errorreporting v0.3.1 // indirect
	cloud.google.com/go/essentialcontacts v1.7.2 // indirect
	cloud.google.com/go/eventarc v1.15.0 // indirect
	cloud.google.com/go/filestore v1.9.2 // indirect
	cloud.google.com/go/firestore v1.17.0 // indirect
	cloud.google.com/go/functions v1.19.2 // indirect
	cloud.google.com/go/gaming v1.10.1 // indirect
	cloud.google.com/go/gkebackup v1.6.2 // indirect
	cloud.google.com/go/gkeconnect v0.12.0 // indirect
	cloud.google.com/go/gkehub v0.15.2 // indirect
	cloud.google.com/go/gkemulticloud v1.4.1 // indirect
	cloud.google.com/go/grafeas v0.3.12 // indirect
	cloud.google.com/go/gsuiteaddons v1.7.2 // indirect
	cloud.google.com/go/iam v1.3.0 // indirect
	cloud.google.com/go/iap v1.10.2 // indirect
	cloud.google.com/go/ids v1.5.2 // indirect
	cloud.google.com/go/iot v1.8.2 // indirect
	cloud.google.com/go/kms v1.20.2 // indirect
	cloud.google.com/go/language v1.14.2 // indirect
	cloud.google.com/go/lifesciences v0.10.2 // indirect
	cloud.google.com/go/logging v1.12.0 // indirect
	cloud.google.com/go/longrunning v0.6.3 // indirect
	cloud.google.com/go/managedidentities v1.7.2 // indirect
	cloud.google.com/go/maps v1.16.0 // indirect
	cloud.google.com/go/mediatranslation v0.9.2 // indirect
	cloud.google.com/go/memcache v1.11.2 // indirect
	cloud.google.com/go/metastore v1.14.2 // indirect
	cloud.google.com/go/networkconnectivity v1.16.0 // indirect
	cloud.google.com/go/networkmanagement v1.17.0 // indirect
	cloud.google.com/go/networksecurity v0.10.2 // indirect
	cloud.google.com/go/notebooks v1.12.2 // indirect
	cloud.google.com/go/optimization v1.7.2 // indirect
	cloud.google.com/go/orchestration v1.11.1 // indirect
	cloud.google.com/go/orgpolicy v1.14.1 // indirect
	cloud.google.com/go/osconfig v1.14.2 // indirect
	cloud.google.com/go/oslogin v1.14.2 // indirect
	cloud.google.com/go/phishingprotection v0.9.2 // indirect
	cloud.google.com/go/policytroubleshooter v1.11.2 // indirect
	cloud.google.com/go/privatecatalog v0.10.2 // indirect
	cloud.google.com/go/pubsub v1.45.3 // indirect
	cloud.google.com/go/pubsublite v1.8.2 // indirect
	cloud.google.com/go/recaptchaenterprise v1.3.1 // indirect
	cloud.google.com/go/recaptchaenterprise/v2 v2.19.1 // indirect
	cloud.google.com/go/recommendationengine v0.9.2 // indirect
	cloud.google.com/go/recommender v1.13.2 // indirect
	cloud.google.com/go/redis v1.17.2 // indirect
	cloud.google.com/go/resourcemanager v1.10.2 // indirect
	cloud.google.com/go/resourcesettings v1.8.2 // indirect
	cloud.google.com/go/retail v1.19.1 // indirect
	cloud.google.com/go/run v1.8.0 // indirect
	cloud.google.com/go/scheduler v1.11.2 // indirect
	cloud.google.com/go/secretmanager v1.14.2 // indirect
	cloud.google.com/go/security v1.18.2 // indirect
	cloud.google.com/go/securitycenter v1.35.2 // indirect
	cloud.google.com/go/servicecontrol v1.14.2 // indirect
	cloud.google.com/go/servicedirectory v1.12.2 // indirect
	cloud.google.com/go/servicemanagement v1.10.2 // indirect
	cloud.google.com/go/serviceusage v1.9.2 // indirect
	cloud.google.com/go/shell v1.8.2 // indirect
	cloud.google.com/go/spanner v1.73.0 // indirect
	cloud.google.com/go/speech v1.25.2 // indirect
	cloud.google.com/go/storagetransfer v1.11.2 // indirect
	cloud.google.com/go/talent v1.7.2 // indirect
	cloud.google.com/go/texttospeech v1.10.0 // indirect
	cloud.google.com/go/tpu v1.7.2 // indirect
	cloud.google.com/go/trace v1.11.2 // indirect
	cloud.google.com/go/translate v1.12.2 // indirect
	cloud.google.com/go/video v1.23.2 // indirect
	cloud.google.com/go/videointelligence v1.12.2 // indirect
	cloud.google.com/go/vision v1.2.0 // indirect
	cloud.google.com/go/vision/v2 v2.9.2 // indirect
	cloud.google.com/go/vmmigration v1.8.2 // indirect
	cloud.google.com/go/vmwareengine v1.3.2 // indirect
	cloud.google.com/go/vpcaccess v1.8.2 // indirect
	cloud.google.com/go/webrisk v1.10.2 // indirect
	cloud.google.com/go/websecurityscanner v1.7.2 // indirect
	cloud.google.com/go/workflows v1.13.2 // indirect
	dario.cat/mergo v1.0.1 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.24 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.13 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/GoogleCloudPlatform/grpc-gcp-go/grpcgcp v1.5.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.25.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.49.0 // indirect
	github.com/GoogleContainerTools/kaniko v1.23.2 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Microsoft/hcsshim v0.12.5 // indirect
	github.com/ProtonMail/go-crypto v1.1.2 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/ahmetalpbalkan/dlog v0.0.0-20170105205344-4fb5f8204f26 // indirect
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/apex/log v1.9.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2 v1.32.4 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.28.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.45 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.36.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.27.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.0 // indirect
	github.com/aws/smithy-go v1.22.1 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20241115173249-4b041aa90387 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chrismellard/docker-credential-acr-env v0.0.0-20230304212654-82a0ddb27589 // indirect
	github.com/cloudflare/circl v1.5.0 // indirect
	github.com/cncf/xds/go v0.0.0-20240905190251-b4127c9b8d78 // indirect
	github.com/containerd/errdefs v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.1 // indirect
	github.com/containerd/ttrpc v1.2.5 // indirect
	github.com/containerd/typeurl/v2 v2.2.3 // indirect
	github.com/cyphar/filepath-securejoin v0.3.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/docker-credential-helpers v0.8.2 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dprotaso/go-yit v0.0.0-20220510233725-9ba8df137936 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/envoyproxy/go-control-plane v0.13.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.1.0 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/gdamore/tcell/v2 v2.7.4 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/runtime v0.28.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/strfmt v0.23.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gobuffalo/here v0.6.0 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.1 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/google/gnostic-models v0.6.9-0.20230804172637-c7be7c783f49 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/martian/v3 v3.3.3 // indirect
	github.com/google/pprof v0.0.0-20241203143554-1e3fdc7de467 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/safetext v0.0.0-20230106111101-7156a760e523 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/cloud-bigtable-clients-test v0.0.2 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/markbates/pkger v0.17.1 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/ioprogress v0.0.0-20180201004757-6a23b12fa88e // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/opencontainers/selinux v1.11.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.20.5 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.60.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/tview v0.0.0-20241103174730-c76f7879f592 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.8.0 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/sigstore/cosign/v2 v2.2.4 // indirect
	github.com/sigstore/rekor v1.3.6 // indirect
	github.com/sigstore/sigstore v1.8.3 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/viper v1.18.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tonistiigi/go-csvvalue v0.0.0-20240814133006-030d3b2625d0 // indirect
	github.com/vbatts/tar-split v0.11.6 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.einride.tech/aip v0.68.0 // indirect
	go.lsp.dev/pkg v0.0.0-20210323044036-f7deec69b52e // indirect
	go.mongodb.org/mongo-driver v1.14.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.32.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.57.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.57.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.32.0 // indirect
	go.uber.org/automaxprocs v1.5.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/exp v0.0.0-20241204233417-43b7b7cde48d // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/net v0.32.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241202173237-19429a94021a // indirect
	google.golang.org/grpc/stats/opentelemetry v0.0.0-20241028142157-ada6787961b3 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	k8s.io/kube-openapi v0.0.0-20231010175941-2dd684a91f00 // indirect
	rsc.io/binaryregexp v0.2.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/kind v0.20.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.3.0 // indirect
)
