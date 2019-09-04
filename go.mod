module github.com/GoogleContainerTools/skaffold

go 1.12

require (
	4d63.com/tz v0.0.0-20190311034157-bd6cee76f731
	cloud.google.com/go v0.43.0
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Microsoft/go-winio v0.4.11 // indirect
	github.com/Netflix/go-expect v0.0.0-20180928190340-9d1f4485533b // indirect
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.1.1
	github.com/containerd/continuity v0.0.0-20181027224239-bea7585dbfac // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/cli v0.0.0-20181026145426-51668a30f262
	github.com/docker/distribution v0.0.0-20180327202408-83389a148052
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/docker-credential-helpers v0.6.1 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.0-20180209012529-399ea8c73916 // indirect
	github.com/docker/go-units v0.3.3 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/protobuf v1.1.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.3.1
	github.com/google/go-containerregistry v0.0.0-20190717132004-e8c6a4993fa7
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/google/uuid v1.1.1
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.6.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.8.5
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/imdario/mergo v0.3.6
	github.com/karrick/godirwalk v1.7.5
	github.com/knative/pkg v0.0.0-20190730155243-972acd413fb9 // indirect
	github.com/krishicks/yaml-patch v0.0.10
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.3.3
	github.com/morikuni/aec v0.0.0-20170113033406-39771216ff4c // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v0.9.1 // indirect
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910 // indirect
	github.com/prometheus/common v0.0.0-20181126121408-4724e9255275 // indirect
	github.com/prometheus/procfs v0.0.0-20181126161756-619930b0b471 // indirect
	github.com/rjeczalik/notify v0.9.2
	github.com/segmentio/textio v1.2.0
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/cobra v0.0.4
	github.com/spf13/pflag v1.0.3
	github.com/tektoncd/pipeline v0.5.1-0.20190731183258-9d7e37e85bf8
	golang.org/x/crypto v0.0.0-20190605123033-f99c8df09eb5
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7 // indirect
	google.golang.org/api v0.7.0
	google.golang.org/genproto v0.0.0-20190716160619-c506a9f90610
	google.golang.org/grpc v1.21.1
	gopkg.in/AlecAivazis/survey.v1 v1.6.1
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/russross/blackfriday.v2 v2.0.1
	gopkg.in/src-d/go-billy.v4 v4.3.0 // indirect
	gopkg.in/src-d/go-git.v4 v4.11.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190620073856-dcce3486da33
	k8s.io/apimachinery v0.0.0-20190620073744-d16981aedf33
	k8s.io/client-go v0.0.0-20190620074045-585a16d2e773
	k8s.io/kubectl v0.0.0-20190622051205-955b067cc6d3
	k8s.io/utils v0.0.0-20190221042446-c2654d5206da
	knative.dev/pkg v0.0.0-20190730155243-972acd413fb9 // indirect
)

replace gopkg.in/russross/blackfriday.v2 v2.0.1 => github.com/russross/blackfriday/v2 v2.0.1
