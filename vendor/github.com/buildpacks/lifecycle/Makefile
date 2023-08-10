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

ACCEPTANCE_TIMEOUT?=2400s
GOCMD?=go
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
LINUX_COMPILATION_IMAGE?=golang:1.16-alpine
WINDOWS_COMPILATION_IMAGE?=golang:1.16-windowsservercore-1809
SOURCE_COMPILATION_IMAGE?=lifecycle-img
BUILD_CTR?=lifecycle-ctr
DOCKER_CMD?=make test

GOFILES := $(shell $(GOCMD) run tools$/lister$/main.go)

all: test build package

build: build-linux-amd64 build-linux-arm64 build-windows-amd64

build-linux-amd64: build-linux-amd64-lifecycle build-linux-amd64-symlinks build-linux-amd64-launcher
build-linux-arm64: build-linux-arm64-lifecycle build-linux-arm64-symlinks build-linux-arm64-launcher
build-windows-amd64: build-windows-amd64-lifecycle build-windows-amd64-symlinks build-windows-amd64-launcher

build-image-linux-amd64: build-linux-amd64 package-linux-amd64
build-image-linux-amd64: ARCHIVE_PATH=$(BUILD_DIR)/lifecycle-v$(LIFECYCLE_VERSION)+linux.x86-64.tgz
build-image-linux-amd64:
	$(GOCMD) run ./tools/image/main.go -daemon -lifecyclePath $(ARCHIVE_PATH) -os linux -arch amd64 -tag lifecycle:$(LIFECYCLE_IMAGE_TAG)

build-image-linux-arm64: build-linux-arm64 package-linux-arm64
build-image-linux-arm64: ARCHIVE_PATH=$(BUILD_DIR)/lifecycle-v$(LIFECYCLE_VERSION)+linux.arm64.tgz
build-image-linux-arm64:
	$(GOCMD) run ./tools/image/main.go -daemon -lifecyclePath $(ARCHIVE_PATH) -os linux -arch arm64 -tag lifecycle:$(LIFECYCLE_IMAGE_TAG)

build-image-windows-amd64: build-windows-amd64 package-windows-amd64
build-image-windows-amd64: ARCHIVE_PATH=$(BUILD_DIR)/lifecycle-v$(LIFECYCLE_VERSION)+windows.x86-64.tgz
build-image-windows-amd64:
	$(GOCMD) run ./tools/image/main.go -daemon -lifecyclePath $(ARCHIVE_PATH) -os windows -arch amd64 -tag lifecycle:$(LIFECYCLE_IMAGE_TAG)

build-linux-amd64-lifecycle: $(BUILD_DIR)/linux-amd64/lifecycle/lifecycle

build-linux-arm64-lifecycle: $(BUILD_DIR)/linux-arm64/lifecycle/lifecycle

$(BUILD_DIR)/linux-amd64/lifecycle/lifecycle: export GOOS:=linux
$(BUILD_DIR)/linux-amd64/lifecycle/lifecycle: export GOARCH:=amd64
$(BUILD_DIR)/linux-amd64/lifecycle/lifecycle: OUT_DIR?=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
$(BUILD_DIR)/linux-amd64/lifecycle/lifecycle: $(GOFILES)
$(BUILD_DIR)/linux-amd64/lifecycle/lifecycle:
	@echo "> Building lifecycle/lifecycle for $(GOOS)/$(GOARCH)..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/lifecycle -a ./cmd/lifecycle

$(BUILD_DIR)/linux-arm64/lifecycle/lifecycle: export GOOS:=linux
$(BUILD_DIR)/linux-arm64/lifecycle/lifecycle: export GOARCH:=arm64
$(BUILD_DIR)/linux-arm64/lifecycle/lifecycle: OUT_DIR?=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
$(BUILD_DIR)/linux-arm64/lifecycle/lifecycle: $(GOFILES)
$(BUILD_DIR)/linux-arm64/lifecycle/lifecycle:
	@echo "> Building lifecycle/lifecycle for $(GOOS)/$(GOARCH)..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/lifecycle -a ./cmd/lifecycle

build-linux-amd64-launcher: $(BUILD_DIR)/linux-amd64/lifecycle/launcher

$(BUILD_DIR)/linux-amd64/lifecycle/launcher: export GOOS:=linux
$(BUILD_DIR)/linux-amd64/lifecycle/launcher: export GOARCH:=amd64
$(BUILD_DIR)/linux-amd64/lifecycle/launcher: OUT_DIR?=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
$(BUILD_DIR)/linux-amd64/lifecycle/launcher: $(GOFILES)
$(BUILD_DIR)/linux-amd64/lifecycle/launcher:
	@echo "> Building lifecycle/launcher for $(GOOS)/$(GOARCH)..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/launcher -a ./cmd/launcher
	test $$(du -m $(OUT_DIR)/launcher|cut -f 1) -le 3

build-linux-arm64-launcher: $(BUILD_DIR)/linux-arm64/lifecycle/launcher

$(BUILD_DIR)/linux-arm64/lifecycle/launcher: export GOOS:=linux
$(BUILD_DIR)/linux-arm64/lifecycle/launcher: export GOARCH:=arm64
$(BUILD_DIR)/linux-arm64/lifecycle/launcher: OUT_DIR?=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
$(BUILD_DIR)/linux-arm64/lifecycle/launcher: $(GOFILES)
$(BUILD_DIR)/linux-arm64/lifecycle/launcher:
	@echo "> Building lifecycle/launcher for $(GOOS)/$(GOARCH)..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/launcher -a ./cmd/launcher
	test $$(du -m $(OUT_DIR)/launcher|cut -f 1) -le 3

build-linux-amd64-symlinks: export GOOS:=linux
build-linux-amd64-symlinks: export GOARCH:=amd64
build-linux-amd64-symlinks: OUT_DIR?=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
build-linux-amd64-symlinks:
	@echo "> Creating phase symlinks for $(GOOS)/$(GOARCH)..."
	ln -sf lifecycle $(OUT_DIR)/detector
	ln -sf lifecycle $(OUT_DIR)/analyzer
	ln -sf lifecycle $(OUT_DIR)/restorer
	ln -sf lifecycle $(OUT_DIR)/builder
	ln -sf lifecycle $(OUT_DIR)/exporter
	ln -sf lifecycle $(OUT_DIR)/rebaser
	ln -sf lifecycle $(OUT_DIR)/creator

build-linux-arm64-symlinks: export GOOS:=linux
build-linux-arm64-symlinks: export GOARCH:=arm64
build-linux-arm64-symlinks: OUT_DIR?=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
build-linux-arm64-symlinks:
	@echo "> Creating phase symlinks for $(GOOS)/$(GOARCH)..."
	ln -sf lifecycle $(OUT_DIR)/detector
	ln -sf lifecycle $(OUT_DIR)/analyzer
	ln -sf lifecycle $(OUT_DIR)/restorer
	ln -sf lifecycle $(OUT_DIR)/builder
	ln -sf lifecycle $(OUT_DIR)/exporter
	ln -sf lifecycle $(OUT_DIR)/rebaser
	ln -sf lifecycle $(OUT_DIR)/creator

build-windows-amd64-lifecycle: $(BUILD_DIR)/windows-amd64/lifecycle/lifecycle.exe

$(BUILD_DIR)/windows-amd64/lifecycle/lifecycle.exe: export GOOS:=windows
$(BUILD_DIR)/windows-amd64/lifecycle/lifecycle.exe: export GOARCH:=amd64
$(BUILD_DIR)/windows-amd64/lifecycle/lifecycle.exe: OUT_DIR?=$(BUILD_DIR)$/$(GOOS)-$(GOARCH)$/lifecycle
$(BUILD_DIR)/windows-amd64/lifecycle/lifecycle.exe: $(GOFILES)
$(BUILD_DIR)/windows-amd64/lifecycle/lifecycle.exe:
	@echo "> Building lifecycle/lifecycle for $(GOOS)/$(GOARCH)..."
	$(GOBUILD) -o $(OUT_DIR)$/lifecycle.exe -a .$/cmd$/lifecycle

build-windows-amd64-launcher: $(BUILD_DIR)/windows-amd64/lifecycle/launcher.exe

$(BUILD_DIR)/windows-amd64/lifecycle/launcher.exe: export GOOS:=windows
$(BUILD_DIR)/windows-amd64/lifecycle/launcher.exe: export GOARCH:=amd64
$(BUILD_DIR)/windows-amd64/lifecycle/launcher.exe: OUT_DIR?=$(BUILD_DIR)$/$(GOOS)-$(GOARCH)$/lifecycle
$(BUILD_DIR)/windows-amd64/lifecycle/launcher.exe: $(GOFILES)
$(BUILD_DIR)/windows-amd64/lifecycle/launcher.exe:
	@echo "> Building lifecycle/launcher for $(GOOS)/$(GOARCH)..."
	$(GOBUILD) -o $(OUT_DIR)$/launcher.exe -a .$/cmd$/launcher

build-windows-amd64-symlinks: export GOOS:=windows
build-windows-amd64-symlinks: export GOARCH:=amd64
build-windows-amd64-symlinks: OUT_DIR?=$(BUILD_DIR)$/$(GOOS)-$(GOARCH)$/lifecycle
build-windows-amd64-symlinks:
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

build-darwin-amd64: build-darwin-amd64-lifecycle build-darwin-amd64-launcher

build-darwin-amd64-lifecycle: $(BUILD_DIR)/darwin-amd64/lifecycle/lifecycle
$(BUILD_DIR)/darwin-amd64/lifecycle/lifecycle: export GOOS:=darwin
$(BUILD_DIR)/darwin-amd64/lifecycle/lifecycle: export GOARCH:=amd64
$(BUILD_DIR)/darwin-amd64/lifecycle/lifecycle: OUT_DIR:=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
$(BUILD_DIR)/darwin-amd64/lifecycle/lifecycle: $(GOFILES)
$(BUILD_DIR)/darwin-amd64/lifecycle/lifecycle:
	@echo "> Building lifecycle for darwin/amd64..."
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/lifecycle -a ./cmd/lifecycle
	@echo "> Creating lifecycle symlinks for darwin/amd64..."
	ln -sf lifecycle $(OUT_DIR)/detector
	ln -sf lifecycle $(OUT_DIR)/analyzer
	ln -sf lifecycle $(OUT_DIR)/restorer
	ln -sf lifecycle $(OUT_DIR)/builder
	ln -sf lifecycle $(OUT_DIR)/exporter
	ln -sf lifecycle $(OUT_DIR)/rebaser

build-darwin-amd64-launcher: $(BUILD_DIR)/darwin-amd64/lifecycle/launcher
$(BUILD_DIR)/darwin-amd64/lifecycle/launcher: export GOOS:=darwin
$(BUILD_DIR)/darwin-amd64/lifecycle/launcher: export GOARCH:=amd64
$(BUILD_DIR)/darwin-amd64/lifecycle/launcher: OUT_DIR:=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
$(BUILD_DIR)/darwin-amd64/lifecycle/launcher: $(GOFILES)
$(BUILD_DIR)/darwin-amd64/lifecycle/launcher:
	@echo "> Building launcher for darwin/amd64..."
	mkdir -p $(OUT_DIR)
	$(GOENV) $(GOBUILD) -o $(OUT_DIR)/launcher -a ./cmd/launcher
	test $$(du -m $(OUT_DIR)/launcher|cut -f 1) -le 4

install-goimports:
	@echo "> Installing goimports..."
	$(GOCMD) install golang.org/x/tools/cmd/goimports@v0.1.2

install-yj:
	@echo "> Installing yj..."
	$(GOCMD) install github.com/sclevine/yj@v0.0.0-20210612025309-737bdf40a5d1

install-mockgen:
	@echo "> Installing mockgen..."
	$(GOCMD) install github.com/golang/mock/mockgen@v1.5.0

install-golangci-lint:
	@echo "> Installing golangci-lint..."
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.42.1

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

tidy:
	@echo "> Tidying modules..."
	$(GOCMD) mod tidy
	cd tools && $(GOCMD) mod tidy

test: unit acceptance

# append coverage arguments
ifeq ($(TEST_COVERAGE), 1)
unit: GOTESTFLAGS:=$(GOTESTFLAGS) -coverprofile=./out/tests/coverage-unit.txt -covermode=atomic
endif
unit: out
unit: UNIT_PACKAGES=$(shell $(GOCMD) list ./... | grep -v acceptance)
unit: format lint tidy install-yj
	@echo "> Running unit tests..."
	$(GOTEST) $(GOTESTFLAGS) -v -count=1 $(UNIT_PACKAGES)

out:
	@mkdir out || (exit 0)
	mkdir out$/tests || (exit 0)

acceptance: format tidy
	@echo "> Running acceptance tests..."
	$(GOTEST) -v -count=1 -tags=acceptance -timeout=$(ACCEPTANCE_TIMEOUT) ./acceptance/...

clean:
	@echo "> Cleaning workspace..."
	rm -rf $(BUILD_DIR)

package: package-linux-amd64 package-linux-arm64 package-windows-amd64

package-linux-amd64: GOOS:=linux
package-linux-amd64: GOARCH:=amd64
package-linux-amd64: INPUT_DIR:=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
package-linux-amd64: ARCHIVE_PATH=$(BUILD_DIR)/lifecycle-v$(LIFECYCLE_VERSION)+$(GOOS).x86-64.tgz
package-linux-amd64: PACKAGER=./tools/packager/main.go
package-linux-amd64:
	@echo "> Packaging lifecycle for $(GOOS)/$(GOARCH)..."
	$(GOCMD) run $(PACKAGER) --inputDir $(INPUT_DIR) -archivePath $(ARCHIVE_PATH) -descriptorPath $(LIFECYCLE_DESCRIPTOR_PATH) -version $(LIFECYCLE_VERSION)

package-linux-arm64: GOOS:=linux
package-linux-arm64: GOARCH:=arm64
package-linux-arm64: INPUT_DIR:=$(BUILD_DIR)/$(GOOS)-$(GOARCH)/lifecycle
package-linux-arm64: ARCHIVE_PATH=$(BUILD_DIR)/lifecycle-v$(LIFECYCLE_VERSION)+$(GOOS).arm64.tgz
package-linux-arm64: PACKAGER=./tools/packager/main.go
package-linux-arm64:
	@echo "> Packaging lifecycle for $(GOOS)/$(GOARCH)..."
	$(GOCMD) run $(PACKAGER) --inputDir $(INPUT_DIR) -archivePath $(ARCHIVE_PATH) -descriptorPath $(LIFECYCLE_DESCRIPTOR_PATH) -version $(LIFECYCLE_VERSION)

package-windows-amd64: GOOS:=windows
package-windows-amd64: GOARCH:=amd64
package-windows-amd64: INPUT_DIR:=$(BUILD_DIR)$/$(GOOS)-$(GOARCH)$/lifecycle
package-windows-amd64: ARCHIVE_PATH=$(BUILD_DIR)$/lifecycle-v$(LIFECYCLE_VERSION)+$(GOOS).x86-64.tgz
package-windows-amd64: PACKAGER=.$/tools$/packager$/main.go
package-windows-amd64:
	@echo "> Packaging lifecycle for $(GOOS)/$(GOARCH)..."
	$(GOCMD) run $(PACKAGER) --inputDir $(INPUT_DIR) -archivePath $(ARCHIVE_PATH) -descriptorPath $(LIFECYCLE_DESCRIPTOR_PATH) -version $(LIFECYCLE_VERSION)

# Ensure workdir is clean and build image from .git
docker-build-source-image-windows: $(GOFILES)
docker-build-source-image-windows:
	$(if $(shell git status --short), @echo Uncommitted changes. Refusing to run. && exit 1)
	docker build .git -f tools/Dockerfile.windows --tag $(SOURCE_COMPILATION_IMAGE) --build-arg image_tag=$(WINDOWS_COMPILATION_IMAGE) --cache-from=$(SOURCE_COMPILATION_IMAGE) --isolation=process --compress

docker-run-windows: docker-build-source-image-windows
docker-run-windows:
	@echo "> Running '$(DOCKER_CMD)' in docker windows..."
	@docker volume rm -f lifecycle-out
	docker run -v lifecycle-out:c:/lifecycle/out -e LIFECYCLE_VERSION -e PLATFORM_API -e BUILDPACK_API -v gopathcache:c:/gopath -v '\\.\pipe\docker_engine:\\.\pipe\docker_engine' --isolation=process --interactive --tty --rm $(SOURCE_COMPILATION_IMAGE) $(DOCKER_CMD)
	docker run -v lifecycle-out:c:/lifecycle/out --rm $(SOURCE_COMPILATION_IMAGE) tar -cf- out | tar -xf-
	@docker volume rm -f lifecycle-out

