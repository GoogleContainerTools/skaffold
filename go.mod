module github.com/GoogleContainerTools/skaffold

go 1.12

require (
	4d63.com/tz v0.0.0-20190311034157-bd6cee76f731
	cloud.google.com/go v0.43.0
	github.com/Azure/go-autorest v12.4.0+incompatible // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20171113091838-e9091a26100e // indirect
	github.com/Microsoft/go-winio v0.4.13 // indirect
	github.com/Netflix/go-expect v0.0.0-20190729225929-0e00d9168667 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/bmatcuk/doublestar v1.1.4
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/cli v0.0.0-20190731011326-e505a7c2169a
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.14.0-0.20190319215453-e7b5f7dbe98c
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.0-20181218153428-b84716841b82 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/evanphx/json-patch v4.5.0+incompatible // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.3.0
	github.com/google/go-containerregistry v0.0.0-20190729175742-ef12d49c8daf
	github.com/google/go-github v17.0.0+incompatible
	github.com/google/go-querystring v1.0.0 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/gophercloud/gophercloud v0.2.0 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.5
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/hinshun/vt10x v0.0.0-20180809195222-d55458df857c // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/json-iterator/go v1.1.7 // indirect
	github.com/karrick/godirwalk v1.10.12
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/krishicks/yaml-patch v0.0.10
	github.com/mattn/go-colorable v0.1.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/moby/buildkit v0.6.0
	github.com/opencontainers/runc v1.0.1-0.20190307181833-2b18fe1d885e // indirect
	github.com/pkg/errors v0.8.1
	github.com/prometheus/common v0.6.0 // indirect
	github.com/prometheus/procfs v0.0.3 // indirect
	github.com/rjeczalik/notify v0.9.2
	github.com/russross/blackfriday v2.0.0+incompatible
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.3
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/sys v0.0.0-20190730183949-1393eb018365 // indirect
	google.golang.org/api v0.7.0
	google.golang.org/genproto v0.0.0-20190716160619-c506a9f90610
	google.golang.org/grpc v1.22.1
	gopkg.in/AlecAivazis/survey.v1 v1.8.5
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.0
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190731142925-739c7f7721ed
	k8s.io/apimachinery v0.0.0-20190731142807-035e418f1ad9
	k8s.io/client-go v0.0.0-20190731143132-de47f833b8db
	k8s.io/klog v0.3.3 // indirect
	k8s.io/kube-openapi v0.0.0-20190722073852-5e22f3d471e6 // indirect
	k8s.io/kubectl v0.0.0-20190731145842-49f722bccf25
	k8s.io/utils v0.0.0-20190712204705-3dccf664f023 // indirect
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v11.1.2+incompatible
	// gopkg.in/russross/blackfriday.v2 v2.0.1 => github.com/russross/blackfriday/v2 v2.0.1
	github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2
)
