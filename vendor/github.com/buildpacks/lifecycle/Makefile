GOCMD?=go
GOENV=GOARCH=amd64 CGO_ENABLED=0
GOBUILD=$(GOCMD) build -mod=vendor -ldflags "-s -w -X 'github.com/buildpacks/lifecycle/cmd.Version=$(LIFECYCLE_VERSION)' -X 'github.com/buildpacks/lifecycle/cmd.SCMRepository=$(SCM_REPO)' -X 'github.com/buildpacks/lifecycle/cmd.SCMCommit=$(SCM_COMMIT)'"
GOTEST=$(GOCMD) test -mod=vendor
LIFECYCLE_VERSION?=0.0.0
PLATFORM_API=0.2
BUILDPACK_API=0.2
SCM_REPO?=
SCM_COMMIT=$$(git rev-parse --short HEAD)
ARCHIVE_NAME=lifecycle-v$(LIFECYCLE_VERSION)+linux.x86-64

define LIFECYCLE_DESCRIPTOR
[api]
  platform = "$(PLATFORM_API)"
  buildpack = "$(BUILDPACK_API)"

[lifecycle]
  version = "$(LIFECYCLE_VERSION)"
endef

all: test build package

build: build-linux

build-macos:
	@echo "> Building for macos..."
	mkdir -p ./out/lifecycle
	GOOS=darwin $(GOENV) $(GOBUILD) -o ./out/lifecycle -a ./cmd/...

build-linux:
	@echo "> Building for linux..."
	mkdir -p ./out/lifecycle
	GOOS=linux $(GOENV) $(GOBUILD) -o ./out/lifecycle -a ./cmd/...

build-windows:
	@echo "> Building for windows..."
	mkdir -p ./out/lifecycle
	GOOS=windows $(GOENV) $(GOBUILD) -o ./out/lifecycle -a ./cmd/...

descriptor: export LIFECYCLE_DESCRIPTOR:=$(LIFECYCLE_DESCRIPTOR)
descriptor:
	@echo "> Writing descriptor file..."
	mkdir -p ./out
	echo "$${LIFECYCLE_DESCRIPTOR}" > ./out/lifecycle.toml

install-goimports:
	@echo "> Installing goimports..."
	cd tools; $(GOCMD) install -mod=vendor golang.org/x/tools/cmd/goimports

install-yj:
	@echo "> Installing yj..."
	cd tools; $(GOCMD) install -mod=vendor github.com/sclevine/yj

install-mockgen:
	@echo "> Installing mockgen..."
	cd tools; $(GOCMD) install -mod=vendor github.com/golang/mock/mockgen

install-golangci-lint:
	@echo "> Installing golangci-lint..."
	cd tools; $(GOCMD) install -mod=vendor github.com/golangci/golangci-lint/cmd/golangci-lint

lint: install-golangci-lint
	@echo "> Linting code..."
	@golangci-lint run -c golangci.yaml

generate: install-mockgen
	@echo "> Generating..."
	$(GOCMD) generate

format: install-goimports
	@echo "> Formating code..."
	test -z $$(goimports -l -w -local github.com/buildpacks/lifecycle $$(find . -type f -name '*.go' -not -path "*/vendor/*"))

test: unit acceptance

unit: format lint install-yj
	@echo "> Running unit tests..."
	$(GOTEST) -v -count=1 ./...

acceptance: format lint
	@echo "> Running acceptance tests..."
	$(GOTEST) -v -count=1 -tags=acceptance ./acceptance/...

clean:
	@echo "> Cleaning workspace..."
	rm -rf ./out

package: descriptor
	@echo "> Packaging lifecycle..."
	tar czf ./out/$(ARCHIVE_NAME).tgz -C out lifecycle.toml lifecycle
