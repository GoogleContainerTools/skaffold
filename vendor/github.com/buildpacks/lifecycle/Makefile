GOCMD?=go
GOFLAGS?=-mod=vendor
GOENV=GOARCH=amd64 CGO_ENABLED=0
LDFLAGS=-s -w
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.Version=$(LIFECYCLE_VERSION)' 
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.SCMRepository=$(SCM_REPO)'
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.SCMCommit=$(SCM_COMMIT)'
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.PlatformAPI=$(PLATFORM_API)'
GOBUILD=$(GOCMD) build -ldflags "$(LDFLAGS)"
GOTEST=$(GOCMD) test
LIFECYCLE_VERSION?=0.0.0
PLATFORM_API?=0.2
BUILDPACK_API=0.2
SCM_REPO?=
SCM_COMMIT?=$$(git rev-parse --short HEAD)
ARCHIVE_NAME=lifecycle-v$(LIFECYCLE_VERSION)+linux.x86-64
BUILD_DIR?=out

export GOFLAGS:=$(GOFLAGS)

define LIFECYCLE_DESCRIPTOR
[api]
  platform = "$(PLATFORM_API)"
  buildpack = "$(BUILDPACK_API)"

[lifecycle]
  version = "$(LIFECYCLE_VERSION)"
endef

all: test build package

build: build-linux

build-darwin:
	@echo "> Building for macos..."
	mkdir -p $(BUILD_DIR)/lifecycle
	GOOS=darwin $(GOENV) $(GOBUILD) -o $(BUILD_DIR)/lifecycle -a ./cmd/...

build-linux:
	@echo "> Building for linux..."
	mkdir -p $(BUILD_DIR)/lifecycle
	GOOS=linux $(GOENV) $(GOBUILD) -o $(BUILD_DIR)/lifecycle -a ./cmd/...

build-windows:
	@echo "> Building for windows..."
	mkdir -p $(BUILD_DIR)/lifecycle
	GOOS=windows $(GOENV) $(GOBUILD) -o $(BUILD_DIR)/lifecycle -a ./cmd/...

descriptor: export LIFECYCLE_DESCRIPTOR:=$(LIFECYCLE_DESCRIPTOR)
descriptor:
	@echo "> Writing descriptor file..."
	mkdir -p $(BUILD_DIR)
	echo "$${LIFECYCLE_DESCRIPTOR}" > $(BUILD_DIR)/lifecycle.toml

install-goimports:
	@echo "> Installing goimports..."
	cd tools; $(GOCMD) install golang.org/x/tools/cmd/goimports

install-yj:
	@echo "> Installing yj..."
	cd tools; $(GOCMD) install github.com/sclevine/yj

install-mockgen:
	@echo "> Installing mockgen..."
	cd tools; $(GOCMD) install github.com/golang/mock/mockgen

install-golangci-lint:
	@echo "> Installing golangci-lint..."
	cd tools; $(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint

lint: install-golangci-lint
	@echo "> Linting code..."
	@golangci-lint run -c golangci.yaml

generate: install-mockgen
	@echo "> Generating..."
	$(GOCMD) generate

format: install-goimports
	@echo "> Formating code..."
	test -z $$(goimports -l -w -local github.com/buildpacks/lifecycle $$(find . -type f -name '*.go' -not -path "*/vendor/*"))

verify-jq:
ifeq (, $(shell which jq))
	$(error "No jq in $$PATH, please install jq")
endif

test: unit acceptance

unit: verify-jq format lint install-yj
	@echo "> Running unit tests..."
	$(GOTEST) -v -count=1 ./...

acceptance: format lint
	@echo "> Running acceptance tests..."
	$(GOTEST) -v -count=1 -tags=acceptance ./acceptance/...

clean:
	@echo "> Cleaning workspace..."
	rm -rf $(BUILD_DIR)

package: descriptor
	@echo "> Packaging lifecycle..."
	tar czf $(BUILD_DIR)/$(ARCHIVE_NAME).tgz -C out lifecycle.toml lifecycle

.PHONY: verify-jq