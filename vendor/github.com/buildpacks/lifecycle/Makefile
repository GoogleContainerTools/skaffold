GOCMD?=go
GOFLAGS?=-mod=vendor
GOARCH?=amd64
GOENV=GOARCH=$(GOARCH) CGO_ENABLED=0
LDFLAGS=-s -w
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.Version=$(LIFECYCLE_VERSION)'
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.SCMRepository=$(SCM_REPO)'
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.SCMCommit=$(SCM_COMMIT)'
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.PlatformAPI=$(PLATFORM_API)'
GOBUILD=go build $(GOFLAGS) -ldflags "$(LDFLAGS)"
GOTEST=$(GOCMD) test $(GOFLAGS)
LIFECYCLE_VERSION?=0.0.0
PLATFORM_API?=0.3
BUILDPACK_API?=0.2
SCM_REPO?=github.com/buildpacks/lifecycle
PARSED_COMMIT:=$(shell git rev-parse --short HEAD)
SCM_COMMIT?=$(PARSED_COMMIT)
BUILD_DIR?=$(PWD)/out
COMPILATION_IMAGE?=golang:1.13-alpine

define LIFECYCLE_DESCRIPTOR
[api]
  platform = "$(PLATFORM_API)"
  buildpack = "$(BUILDPACK_API)"

[lifecycle]
  version = "$(LIFECYCLE_VERSION)"
endef

all: test build package

build: build-linux build-windows

build-linux-lifecycle: export GOOS:=linux
build-linux-lifecycle: OUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
build-linux-lifecycle: GOENV:=GOARCH=$(GOARCH) CGO_ENABLED=1
build-linux-lifecycle: DOCKER_RUN=docker run --workdir=/lifecycle -v $(OUT_DIR):/out -v $(PWD):/lifecycle $(COMPILATION_IMAGE)
build-linux-lifecycle:
	@echo "> Building lifecycle/lifecycle for linux..."
	mkdir -p $(OUT_DIR)
	$(DOCKER_RUN) sh -c 'apk add build-base && $(GOENV) $(GOBUILD) -o /out/lifecycle -a ./cmd/lifecycle'


build-linux-launcher: export GOOS:=linux
build-linux-launcher: OUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
build-linux-launcher:
	@echo "> Building lifecycle/launcher for linux..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/launcher -a ./cmd/launcher
	test $$(du -m $(OUT_DIR)/launcher|cut -f 1) -le 3

build-linux-symlinks: export GOOS:=linux
build-linux-symlinks: OUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
build-linux-symlinks:
	@echo "> Creating phase symlinks for linux..."
	ln -sf lifecycle $(OUT_DIR)/detector
	ln -sf lifecycle $(OUT_DIR)/analyzer
	ln -sf lifecycle $(OUT_DIR)/restorer
	ln -sf lifecycle $(OUT_DIR)/builder
	ln -sf lifecycle $(OUT_DIR)/exporter
	ln -sf lifecycle $(OUT_DIR)/rebaser
	ln -sf lifecycle $(OUT_DIR)/creator

build-linux: build-linux-lifecycle build-linux-symlinks build-linux-launcher

build-windows: export GOOS:=windows
build-windows: OUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
build-windows:
	@echo "> Building for windows..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/launcher -a ./cmd/launcher
	test $$(du -m $(OUT_DIR)/launcher|cut -f 1) -le 3
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/lifecycle.exe -a ./cmd/lifecycle
	ln -sf lifecycle.exe $(OUT_DIR)/analyzer.exe
	ln -sf lifecycle.exe $(OUT_DIR)/restorer.exe
	ln -sf lifecycle.exe $(OUT_DIR)/builder.exe
	ln -sf lifecycle.exe $(OUT_DIR)/exporter.exe
	ln -sf lifecycle.exe $(OUT_DIR)/rebaser.exe
	ln -sf lifecycle.exe $(OUT_DIR)/creator.exe

build-darwin: export GOOS:=darwin
build-darwin: OUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
build-darwin:
	@echo "> Building for macos..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/launcher -a ./cmd/launcher
	test $$(du -m $(OUT_DIR)/launcher|cut -f 1) -le 3
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/lifecycle -a ./cmd/lifecycle
	ln -sf lifecycle $(OUT_DIR)/detector
	ln -sf lifecycle $(OUT_DIR)/analyzer
	ln -sf lifecycle $(OUT_DIR)/restorer
	ln -sf lifecycle $(OUT_DIR)/builder
	ln -sf lifecycle $(OUT_DIR)/exporter
	ln -sf lifecycle $(OUT_DIR)/rebaser

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
	$(GOCMD) generate ./launch

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
	
acceptance-darwin: format lint
	@echo "> Running acceptance tests..."
	$(GOTEST) -v -count=1 -tags=acceptance ./acceptance/...

clean:
	@echo "> Cleaning workspace..."
	rm -rf $(BUILD_DIR)

package: package-linux package-windows

package-linux: export LIFECYCLE_DESCRIPTOR:=$(LIFECYCLE_DESCRIPTOR)
package-linux: GOOS:=linux
package-linux: GOOS_DIR:=$(BUILD_DIR)/$(GOOS)
package-linux: ARCHIVE_NAME=lifecycle-v$(LIFECYCLE_VERSION)+$(GOOS).x86-64
package-linux:
	@echo "> Writing descriptor file for $(GOOS)..."
	mkdir -p $(GOOS_DIR)
	echo "$${LIFECYCLE_DESCRIPTOR}" > $(GOOS_DIR)/lifecycle.toml

	@echo "> Packaging lifecycle for $(GOOS)..."
	tar czf $(BUILD_DIR)/$(ARCHIVE_NAME).tgz -C $(GOOS_DIR) lifecycle.toml lifecycle

package-windows: export LIFECYCLE_DESCRIPTOR:=$(LIFECYCLE_DESCRIPTOR)
package-windows: GOOS:=windows
package-windows: GOOS_DIR:=$(BUILD_DIR)/$(GOOS)
package-windows: ARCHIVE_NAME=lifecycle-v$(LIFECYCLE_VERSION)+$(GOOS).x86-64
package-windows:
	@echo "> Writing descriptor file for $(GOOS)..."
	mkdir -p $(GOOS_DIR)
	echo "$${LIFECYCLE_DESCRIPTOR}" > $(GOOS_DIR)/lifecycle.toml

	@echo "> Packaging lifecycle for $(GOOS)..."
	tar czf $(BUILD_DIR)/$(ARCHIVE_NAME).tgz -C $(GOOS_DIR) lifecycle.toml lifecycle

.PHONY: verify-jq
