ifeq ($(OS),Windows_NT)
SHELL:=cmd.exe
PWD?=$(subst /,\,${CURDIR})
LDFLAGS=-s -w
BLANK:=
/:=\$(BLANK)
else
/:=/
endif

PARSED_COMMIT:=$(shell git rev-parse --short HEAD)

ifeq ($(LIFECYCLE_VERSION),)
LIFECYCLE_VERSION:=$(shell go run tools/version/main.go)
LIFECYCLE_IMAGE_TAG?=$(PARSED_COMMIT)
else
LIFECYCLE_IMAGE_TAG?=$(LIFECYCLE_VERSION)
endif

GOCMD?=go
GOARCH?=amd64
GOENV=GOARCH=$(GOARCH) CGO_ENABLED=0
LIFECYCLE_DESCRIPTOR_PATH?=lifecycle.toml
SCM_REPO?=github.com/buildpacks/lifecycle
SCM_COMMIT?=$(PARSED_COMMIT)
LDFLAGS=-s -w
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.SCMRepository=$(SCM_REPO)'
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.SCMCommit=$(SCM_COMMIT)'
LDFLAGS+=-X 'github.com/buildpacks/lifecycle/cmd.Version=$(LIFECYCLE_VERSION)'
GOBUILD:=go build $(GOFLAGS) -ldflags "$(LDFLAGS)"
GOTEST=$(GOCMD) test $(GOFLAGS)
BUILD_DIR?=$(PWD)$/out
LINUX_COMPILATION_IMAGE?=golang:1.15-alpine
WINDOWS_COMPILATION_IMAGE?=golang:1.15-windowsservercore-1809
SOURCE_COMPILATION_IMAGE?=lifecycle-img
BUILD_CTR?=lifecycle-ctr
DOCKER_CMD?=make test

GOFILES := $(shell $(GOCMD) run tools$/lister$/main.go)

all: test build package

build: build-linux build-windows

build-linux: build-linux-lifecycle build-linux-symlinks build-linux-launcher
build-windows: build-windows-lifecycle build-windows-symlinks build-windows-launcher

build-image-linux: build-linux package-linux
build-image-linux: ARCHIVE_PATH=$(BUILD_DIR)/lifecycle-v$(LIFECYCLE_VERSION)+linux.x86-64.tgz
build-image-linux:
	$(GOCMD) run ./tools/image/main.go -daemon -lifecyclePath $(ARCHIVE_PATH) -os linux -tag lifecycle:$(LIFECYCLE_IMAGE_TAG)

build-image-windows: build-windows package-windows
build-image-windows: ARCHIVE_PATH=$(BUILD_DIR)/lifecycle-v$(LIFECYCLE_VERSION)+windows.x86-64.tgz
build-image-windows:
	$(GOCMD) run ./tools/image/main.go -daemon -lifecyclePath $(ARCHIVE_PATH) -os windows -tag lifecycle:$(LIFECYCLE_IMAGE_TAG)

build-linux-lifecycle: $(BUILD_DIR)/linux/lifecycle/lifecycle

docker-compilation-image-linux:
	docker build ./tools --build-arg from_image=$(LINUX_COMPILATION_IMAGE) --tag $(SOURCE_COMPILATION_IMAGE)


$(BUILD_DIR)/linux/lifecycle/lifecycle: export GOOS:=linux
$(BUILD_DIR)/linux/lifecycle/lifecycle: OUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
$(BUILD_DIR)/linux/lifecycle/lifecycle: GOENV:=GOARCH=$(GOARCH) CGO_ENABLED=1
$(BUILD_DIR)/linux/lifecycle/lifecycle: docker-compilation-image-linux
$(BUILD_DIR)/linux/lifecycle/lifecycle: $(GOFILES)
$(BUILD_DIR)/linux/lifecycle/lifecycle:
	@echo "> Building lifecycle/lifecycle for linux..."
	mkdir -p $(OUT_DIR)
	docker run \
	  --workdir=/lifecycle \
	  --volume $(OUT_DIR):/out \
	  --volume $(PWD):/lifecycle \
	  --volume gocache:/go \
	  $(SOURCE_COMPILATION_IMAGE) \
	  sh -c '$(GOENV) $(GOBUILD) -o /out/lifecycle -a ./cmd/lifecycle'

build-linux-launcher: $(BUILD_DIR)/linux/lifecycle/launcher

$(BUILD_DIR)/linux/lifecycle/launcher: export GOOS:=linux
$(BUILD_DIR)/linux/lifecycle/launcher: OUT_DIR?=$(BUILD_DIR)/$(GOOS)/lifecycle
$(BUILD_DIR)/linux/lifecycle/launcher: $(GOFILES)
$(BUILD_DIR)/linux/lifecycle/launcher:
	@echo "> Building lifecycle/launcher for linux..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/launcher -a ./cmd/launcher
	test $$(du -m $(OUT_DIR)/launcher|cut -f 1) -le 3

build-linux-symlinks: export GOOS:=linux
build-linux-symlinks: OUT_DIR?=$(BUILD_DIR)/$(GOOS)/lifecycle
build-linux-symlinks:
	@echo "> Creating phase symlinks for linux..."
	ln -sf lifecycle $(OUT_DIR)/detector
	ln -sf lifecycle $(OUT_DIR)/analyzer
	ln -sf lifecycle $(OUT_DIR)/restorer
	ln -sf lifecycle $(OUT_DIR)/builder
	ln -sf lifecycle $(OUT_DIR)/exporter
	ln -sf lifecycle $(OUT_DIR)/rebaser
	ln -sf lifecycle $(OUT_DIR)/creator

build-windows-lifecycle: $(BUILD_DIR)/windows/lifecycle/lifecycle.exe

$(BUILD_DIR)/windows/lifecycle/lifecycle.exe: export GOOS:=windows
$(BUILD_DIR)/windows/lifecycle/lifecycle.exe: OUT_DIR?=$(BUILD_DIR)$/$(GOOS)$/lifecycle
$(BUILD_DIR)/windows/lifecycle/lifecycle.exe: $(GOFILES)
$(BUILD_DIR)/windows/lifecycle/lifecycle.exe:
	@echo "> Building lifecycle/lifecycle for Windows..."
	$(GOBUILD) -o $(OUT_DIR)$/lifecycle.exe -a .$/cmd$/lifecycle

build-windows-launcher: $(BUILD_DIR)/windows/lifecycle/launcher.exe

$(BUILD_DIR)/windows/lifecycle/launcher.exe: export GOOS:=windows
$(BUILD_DIR)/windows/lifecycle/launcher.exe: OUT_DIR?=$(BUILD_DIR)$/$(GOOS)$/lifecycle
$(BUILD_DIR)/windows/lifecycle/launcher.exe: $(GOFILES)
$(BUILD_DIR)/windows/lifecycle/launcher.exe:
	@echo "> Building lifecycle/launcher for Windows..."
	$(GOBUILD) -o $(OUT_DIR)$/launcher.exe -a .$/cmd$/launcher

build-windows-symlinks: export GOOS:=windows
build-windows-symlinks: OUT_DIR?=$(BUILD_DIR)$/$(GOOS)$/lifecycle
build-windows-symlinks:
	@echo "> Creating phase symlinks for Windows..."
ifeq ($(OS),Windows_NT)
	call del $(OUT_DIR)$/detector.exe
	call del $(OUT_DIR)$/analyzer.exe
	call del $(OUT_DIR)$/restorer.exe
	call del $(OUT_DIR)$/builder.exe
	call del $(OUT_DIR)$/exporter.exe
	call del $(OUT_DIR)$/rebaser.exe
	call del $(OUT_DIR)$/creator.exe
	call mklink $(OUT_DIR)$/detector.exe lifecycle.exe
	call mklink $(OUT_DIR)$/analyzer.exe lifecycle.exe
	call mklink $(OUT_DIR)$/restorer.exe lifecycle.exe
	call mklink $(OUT_DIR)$/builder.exe  lifecycle.exe
	call mklink $(OUT_DIR)$/exporter.exe lifecycle.exe
	call mklink $(OUT_DIR)$/rebaser.exe  lifecycle.exe
	call mklink $(OUT_DIR)$/creator.exe  lifecycle.exe
else
	ln -sf lifecycle.exe $(OUT_DIR)$/detector.exe
	ln -sf lifecycle.exe $(OUT_DIR)$/analyzer.exe
	ln -sf lifecycle.exe $(OUT_DIR)$/restorer.exe
	ln -sf lifecycle.exe $(OUT_DIR)$/builder.exe
	ln -sf lifecycle.exe $(OUT_DIR)$/exporter.exe
	ln -sf lifecycle.exe $(OUT_DIR)$/rebaser.exe
	ln -sf lifecycle.exe $(OUT_DIR)$/creator.exe
endif

build-darwin: build-darwin-lifecycle build-darwin-launcher

build-darwin-lifecycle: $(BUILD_DIR)/darwin/lifecycle/lifecycle
$(BUILD_DIR)/darwin/lifecycle/lifecycle: export GOOS:=darwin
$(BUILD_DIR)/darwin/lifecycle/lifecycle: OUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
$(BUILD_DIR)/darwin/lifecycle/lifecycle: $(GOFILES)
$(BUILD_DIR)/darwin/lifecycle/lifecycle:
	@echo "> Building lifecycle for macos..."
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/lifecycle -a ./cmd/lifecycle
	@echo "> Creating lifecycle symlinks for macos..."
	ln -sf lifecycle $(OUT_DIR)/detector
	ln -sf lifecycle $(OUT_DIR)/analyzer
	ln -sf lifecycle $(OUT_DIR)/restorer
	ln -sf lifecycle $(OUT_DIR)/builder
	ln -sf lifecycle $(OUT_DIR)/exporter
	ln -sf lifecycle $(OUT_DIR)/rebaser

build-darwin-launcher: $(BUILD_DIR)/darwin/lifecycle/launcher
$(BUILD_DIR)/darwin/lifecycle/launcher: export GOOS:=darwin
$(BUILD_DIR)/darwin/lifecycle/launcher: OUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
$(BUILD_DIR)/darwin/lifecycle/launcher: $(GOFILES)
$(BUILD_DIR)/darwin/lifecycle/launcher:
	@echo "> Building launcher for macos..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/launcher -a ./cmd/launcher
	test $$(du -m $(OUT_DIR)/launcher|cut -f 1) -le 4

install-goimports:
	@echo "> Installing goimports..."
	cd tools && $(GOCMD) install golang.org/x/tools/cmd/goimports

install-yj:
	@echo "> Installing yj..."
	cd tools && $(GOCMD) install github.com/sclevine/yj

install-mockgen:
	@echo "> Installing mockgen..."
	cd tools && $(GOCMD) install github.com/golang/mock/mockgen

install-golangci-lint:
	@echo "> Installing golangci-lint..."
	cd tools && $(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint

lint: install-golangci-lint
	@echo "> Linting code..."
	@golangci-lint run -c golangci.yaml

generate: install-mockgen
	@echo "> Generating..."
	$(GOCMD) generate
	$(GOCMD) generate ./launch

format: install-goimports
	@echo "> Formating code..."
	$(if $(shell goimports -l -w -local github.com/buildpacks/lifecycle .), @echo Fixed formatting errors. Re-run && exit 1)

test: unit acceptance

unit: UNIT_PACKAGES=$(shell $(GOCMD) list ./... | grep -v acceptance)
unit: format lint install-yj
	@echo "> Running unit tests..."
	$(GOTEST) -v -count=1 $(UNIT_PACKAGES)

acceptance: format lint
	@echo "> Running acceptance tests..."
	$(GOTEST) -v -count=1 -tags=acceptance ./acceptance/...

clean:
	@echo "> Cleaning workspace..."
	rm -rf $(BUILD_DIR)

package: package-linux package-windows

package-linux: GOOS:=linux
package-linux: INPUT_DIR:=$(BUILD_DIR)/$(GOOS)/lifecycle
package-linux: ARCHIVE_PATH=$(BUILD_DIR)/lifecycle-v$(LIFECYCLE_VERSION)+$(GOOS).x86-64.tgz
package-linux: PACKAGER=./tools/packager/main.go
package-linux:
	@echo "> Packaging lifecycle for $(GOOS)..."
	$(GOCMD) run $(PACKAGER) --inputDir $(INPUT_DIR) -archivePath $(ARCHIVE_PATH) -descriptorPath $(LIFECYCLE_DESCRIPTOR_PATH) -version $(LIFECYCLE_VERSION)

package-windows: GOOS:=windows
package-windows: INPUT_DIR:=$(BUILD_DIR)$/$(GOOS)$/lifecycle
package-windows: ARCHIVE_PATH=$(BUILD_DIR)$/lifecycle-v$(LIFECYCLE_VERSION)+$(GOOS).x86-64.tgz
package-windows: PACKAGER=.$/tools$/packager$/main.go
package-windows:
	@echo "> Packaging lifecycle for $(GOOS)..."
	$(GOCMD) run $(PACKAGER) --inputDir $(INPUT_DIR) -archivePath $(ARCHIVE_PATH) -descriptorPath $(LIFECYCLE_DESCRIPTOR_PATH) -version $(LIFECYCLE_VERSION)

# Ensure workdir is clean and build image from .git
docker-build-source-image-windows: $(GOFILES)
docker-build-source-image-windows:
	$(if $(shell git status --short), @echo Uncommitted changes. Refusing to run. && exit 1)
	docker build -f tools/Dockerfile.windows --tag $(SOURCE_COMPILATION_IMAGE) --build-arg image_tag=$(WINDOWS_COMPILATION_IMAGE) --cache-from=$(SOURCE_COMPILATION_IMAGE) --isolation=process --quiet .git

docker-run-windows: docker-build-source-image-windows
docker-run-windows:
	@echo "> Running '$(DOCKER_CMD)' in docker windows..."
	@docker volume rm -f lifecycle-out
	docker run -v lifecycle-out:c:/lifecycle/out -e LIFECYCLE_VERSION -e PLATFORM_API -e BUILDPACK_API -v gopathcache:c:/gopath -v '\\.\pipe\docker_engine:\\.\pipe\docker_engine' --isolation=process --interactive --tty --rm $(SOURCE_COMPILATION_IMAGE) $(DOCKER_CMD)
	docker run -v lifecycle-out:c:/lifecycle/out --rm $(SOURCE_COMPILATION_IMAGE) tar -cf- out | tar -xf-
	@docker volume rm -f lifecycle-out

