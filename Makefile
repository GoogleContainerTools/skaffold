# Copyright 2019 The Skaffold Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(or $(shell go env GOBIN),$(GOPATH)/bin)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BUILD_DIR ?= ./out
ORG = github.com/GoogleContainerTools
PROJECT = skaffold
REPOPATH ?= $(ORG)/$(PROJECT)
RELEASE_BUCKET ?= $(PROJECT)
GSC_BUILD_PATH ?= gs://$(RELEASE_BUCKET)/builds/$(COMMIT)
GSC_BUILD_LATEST ?= gs://$(RELEASE_BUCKET)/builds/latest
GSC_LTS_BUILD_PATH ?= gs://$(RELEASE_BUCKET)/lts/builds/$(COMMIT)
GSC_LTS_BUILD_LATEST ?= gs://$(RELEASE_BUCKET)/lts/builds/latest
GSC_LTS_RELEASE_PATH ?= gs://$(RELEASE_BUCKET)/lts/releases/$(VERSION)
GSC_LTS_RELEASE_LATEST ?= gs://$(RELEASE_BUCKET)/lts/releases/latest
GSC_RELEASE_PATH ?= gs://$(RELEASE_BUCKET)/releases/$(VERSION)
GSC_RELEASE_LATEST ?= gs://$(RELEASE_BUCKET)/releases/latest

GCP_ONLY ?= false
GCP_PROJECT ?= k8s-skaffold
GKE_CLUSTER_NAME ?= integration-tests
GKE_ZONE ?= us-central1-a

SUPPORTED_PLATFORMS = linux-amd64 darwin-amd64 windows-amd64.exe linux-arm64 darwin-arm64
BUILD_PACKAGE = $(REPOPATH)/v2/cmd/skaffold

SKAFFOLD_TEST_PACKAGES = ./pkg/skaffold/... ./cmd/... ./hack/... ./pkg/webhook/...
GO_FILES = $(shell find . -type f -name '*.go' -not -path "./pkg/diag/*")

VERSION_PACKAGE = $(REPOPATH)/v2/pkg/skaffold/version
COMMIT = $(shell git rev-parse HEAD)

ifeq "$(strip $(VERSION))" ""
	override VERSION = $(shell git describe --always --tags --dirty)
endif

DATE_FMT = +%Y-%m-%dT%H:%M:%SZ
ifdef SOURCE_DATE_EPOCH
    BUILD_DATE ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u -r "$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u "$(DATE_FMT)")
else
    BUILD_DATE ?= $(shell date "$(DATE_FMT)")
endif

GO_LDFLAGS = -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -X $(VERSION_PACKAGE).buildDate=$(BUILD_DATE)
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitCommit=$(COMMIT)
GO_LDFLAGS += -s -w

GO_BUILD_TAGS = timetzdata

GO_BUILD_TAGS_linux = osusergo netgo static_build release
LDFLAGS_linux = -static

GO_BUILD_TAGS_windows = release

GO_BUILD_TAGS_darwin = release

ifneq "$(strip $(LOCAL))" "true"
	override EMBEDDED_FILES_CHECK = fs/assets/check.txt
endif

# when build for local development (`LOCAL=true make install` can skip license check)
$(BUILD_DIR)/$(PROJECT): $(EMBEDDED_FILES_CHECK) $(GO_FILES) $(BUILD_DIR)
	$(eval ldflags = $(GO_LDFLAGS) $(patsubst %,-extldflags \"%\",$(LDFLAGS_$(GOOS))))
	$(eval tags = $(GO_BUILD_TAGS) $(GO_BUILD_TAGS_$(GOOS)) $(GO_BUILD_TAGS_$(GOOS)_$(GOARCH)))
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 \
	    go build -mod="vendor" -gcflags="all=-N -l" -tags "$(tags)" -ldflags "$(ldflags)" -o $@ $(BUILD_PACKAGE)
ifeq ($(GOOS),darwin)
	codesign --force --deep --sign - $@
endif

.PHONY: install
install: $(BUILD_DIR)/$(PROJECT)
	mkdir -p $(GOPATH)/bin
	rm -f $(GOBIN)/$(PROJECT)
	cp $(BUILD_DIR)/$(PROJECT) $(GOBIN)/$(PROJECT)

.PRECIOUS: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

.PHONY: cross
cross: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

$(BUILD_DIR)/$(PROJECT)-%: $(EMBEDDED_FILES_CHECK) $(GO_FILES) $(BUILD_DIR)
	$(eval os = $(firstword $(subst -, ,$*)))
	$(eval arch = $(lastword $(subst -, ,$(subst .exe,,$*))))
	$(eval ldflags = $(GO_LDFLAGS) $(patsubst %,-extldflags \"%\",$(LDFLAGS_$(os))))
	$(eval tags = $(GO_BUILD_TAGS) $(GO_BUILD_TAGS_$(os)) $(GO_BUILD_TAGS_$(os)_$(arch)))
	GOOS=$(os) GOARCH=$(arch) CGO_ENABLED=1 go build -mod="vendor" -tags "$(tags)" -ldflags "$(ldflags)" -o $@ ./cmd/skaffold
	(cd `dirname $@`; shasum -a 256 `basename $@`) | tee $@.sha256
	file $@ || true

.PHONY: $(BUILD_DIR)/VERSION
$(BUILD_DIR)/VERSION: $(BUILD_DIR)
	@ echo $(VERSION) > $@

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: test
test: $(BUILD_DIR)
	@ ./hack/gotest.sh -count=1 -race -short -timeout=90s $(SKAFFOLD_TEST_PACKAGES)
	@ ./hack/checks.sh
	@ ./hack/linters.sh

.PHONY: unit-tests
unit-tests: $(BUILD_DIR)
	@ ./hack/gotest.sh -count=1 -race -short -timeout=90s $(SKAFFOLD_TEST_PACKAGES)

.PHONY: coverage
coverage: $(BUILD_DIR)
    # https://go-review.git.corp.google.com/c/go/+/569575
	@ ./hack/gotest.sh -count=1 -race -cover -short -timeout=90s -coverprofile=out/coverage.txt -coverpkg="./pkg/...,./cmd/..." $(SKAFFOLD_TEST_PACKAGES)
	@- curl -s https://codecov.io/bash > $(BUILD_DIR)/upload_coverage && bash $(BUILD_DIR)/upload_coverage

.PHONY: checks
checks: $(BUILD_DIR)
	@ ./hack/checks.sh

.PHONY: linters
linters: $(BUILD_DIR)
	@ ./hack/linters.sh

.PHONY: quicktest
quicktest:
	@ ./hack/gotest.sh -short -timeout=60s $(SKAFFOLD_TEST_PACKAGES)

.PHONY: integration-tests
integration-tests:
ifeq ($(GCP_ONLY),true)
	gcloud container clusters get-credentials \
		$(GKE_CLUSTER_NAME) \
		--zone $(GKE_ZONE) \
		--project $(GCP_PROJECT)
	gcloud auth configure-docker us-central1-docker.pkg.dev
endif
	@ GCP_ONLY=$(GCP_ONLY) GKE_CLUSTER_NAME=$(GKE_CLUSTER_NAME) ./hack/gotest.sh -v $(REPOPATH)/v2/integration -timeout 50m $(INTEGRATION_TEST_ARGS)

.PHONY: integration
integration: install integration-tests

.PHONY: release
release: $(BUILD_DIR)/VERSION
	docker build \
		--build-arg VERSION=$(VERSION) \
		-f deploy/skaffold/Dockerfile \
		--target release \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(VERSION) \
                -t gcr.io/$(GCP_PROJECT)/skaffold:latest \
		.

.PHONY: release-build
release-build:
	docker build \
		-f deploy/skaffold/Dockerfile \
		--target release \
		-t gcr.io/$(GCP_PROJECT)/skaffold:edge \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(COMMIT) \
		.

.PHONY: release-lts
release-lts: $(BUILD_DIR)/VERSION
	docker build \
		--build-arg VERSION=$(VERSION) \
		-f deploy/skaffold/Dockerfile.lts \
		--target release \
		-t gcr.io/$(GCP_PROJECT)/skaffold:lts \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(VERSION)-lts \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(SCANNING_MARKER)-lts \
		.

.PHONY: release-lts-build
release-lts-build:
	docker build \
		-f deploy/skaffold/Dockerfile.lts \
		--target release \
		-t gcr.io/$(GCP_PROJECT)/skaffold:edge-lts \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(COMMIT)-lts \
		.

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) hack/bin $(EMBEDDED_FILES_CHECK) fs/assets/schemas_generated/

.PHONY: build_deps
build_deps:
	$(eval DEPS_DIGEST := $(shell ./hack/skaffold-deps-sha1.sh))
	docker build \
		-f deploy/skaffold/Dockerfile.deps \
		-t gcr.io/$(GCP_PROJECT)/build_deps:$(DEPS_DIGEST) \
		deploy/skaffold
	docker push gcr.io/$(GCP_PROJECT)/build_deps:$(DEPS_DIGEST)

skaffold-builder-ci:
	docker build \
		--cache-from gcr.io/$(GCP_PROJECT)/build_deps \
		-f deploy/skaffold/Dockerfile.deps \
		-t gcr.io/$(GCP_PROJECT)/build_deps \
		.
	time docker build \
		-f deploy/skaffold/Dockerfile \
		--target builder \
		-t gcr.io/$(GCP_PROJECT)/skaffold-builder \
		.

.PHONY: skaffold-builder
skaffold-builder:
	time docker build \
		-f deploy/skaffold/Dockerfile \
		--target builder \
		-t gcr.io/$(GCP_PROJECT)/skaffold-builder \
		.

.PHONY: integration-in-kind
integration-in-kind: skaffold-builder
	echo '{}' > /tmp/docker-config
	docker network inspect kind >/dev/null 2>&1 || docker network create kind
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.gradle:/root/.gradle \
		-v $(HOME)/.cache:/root/.cache \
		-v /tmp/docker-config:/root/.docker/config.json \
		-v $(CURDIR)/hack/maven/settings.xml:/root/.m2/settings.xml \
		-e KUBECONFIG=/tmp/kind-config \
		-e INTEGRATION_TEST_ARGS=$(INTEGRATION_TEST_ARGS) \
		-e IT_PARTITION=$(IT_PARTITION) \
		--network kind \
		gcr.io/$(GCP_PROJECT)/skaffold-builder \
		sh -eu -c ' \
			if ! kind get clusters | grep -q kind; then \
			  trap "kind delete cluster" 0 1 2 15; \
			  sh hack/generate-kind-config.sh > /tmp/kind-config.yaml; \
			  cat /tmp/kind-config.yaml; \
			  TERM=dumb kind create cluster --config /tmp/kind-config.yaml; \
			fi; \
			kind get kubeconfig --internal > /tmp/kind-config; \
			make integration \
		'

.PHONY: integration-in-k3d
integration-in-k3d: skaffold-builder
	echo '{}' > /tmp/docker-config
	docker pull rancher/k3d-tools:v3.4.0
	docker run --rm \
		--network="host" \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.gradle:/root/.gradle \
		-v $(HOME)/.cache:/root/.cache \
		-v /tmp/docker-config:/root/.docker/config.json \
		-v $(CURDIR)/hack/maven/settings.xml:/root/.m2/settings.xml \
		-e INTEGRATION_TEST_ARGS=$(INTEGRATION_TEST_ARGS) \
		-e IT_PARTITION=$(IT_PARTITION) \
		gcr.io/$(GCP_PROJECT)/skaffold-builder \
		sh -eu -c ' \
			if ! k3d cluster list | grep -q k3s-default; then \
			  trap "k3d cluster delete" 0 1 2 15; \
			  mkdir -p /tmp/k3d; \
			  sh hack/generate-k3d-registries.sh > /tmp/k3d/registries.yaml; \
			  cat /tmp/k3d/registries.yaml; \
			  TERM=dumb k3d cluster create --verbose \
			      --network k3d \
			      --volume /tmp/k3d:/etc/rancher/k3s; \
			fi; \
			make integration \
		'

.PHONY: integration-in-docker
integration-in-docker: skaffold-builder-ci
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.config/gcloud:/root/.config/gcloud \
		-v $(GOOGLE_APPLICATION_CREDENTIALS):$(GOOGLE_APPLICATION_CREDENTIALS) \
		-v $(CURDIR)/hack/maven/settings.xml:/root/.m2/settings.xml \
		-e GCP_ONLY=$(GCP_ONLY) \
		-e GCP_PROJECT=$(GCP_PROJECT) \
		-e GKE_CLUSTER_NAME=$(GKE_CLUSTER_NAME) \
		-e GKE_ZONE=$(GKE_ZONE) \
		-e DOCKER_CONFIG=/root/.docker \
		-e GOOGLE_APPLICATION_CREDENTIALS=$(GOOGLE_APPLICATION_CREDENTIALS) \
		-e INTEGRATION_TEST_ARGS=$(INTEGRATION_TEST_ARGS) \
		-e IT_PARTITION=$(IT_PARTITION) \
		gcr.io/$(GCP_PROJECT)/skaffold-builder \
		make integration-tests

.PHONY: submit-build-trigger
submit-build-trigger:
	gcloud builds submit . \
		--config=deploy/cloudbuild.yaml \
		--substitutions="_RELEASE_BUCKET=$(RELEASE_BUCKET),COMMIT_SHA=$(COMMIT)"

.PHONY: submit-release-trigger
submit-release-trigger:
	gcloud builds submit . \
		--config=deploy/cloudbuild-release.yaml \
		--substitutions="_RELEASE_BUCKET=$(RELEASE_BUCKET),TAG_NAME=$(VERSION)"

# utilities for skaffold site - not used anywhere else

.PHONY: preview-docs
preview-docs:
	./deploy/docs-v1/local-preview.sh hugo serve -D --bind=0.0.0.0 --ignoreCache

.PHONY: build-docs-preview
build-docs-preview:
	./deploy/docs-v1/local-preview.sh hugo --baseURL=https://skaffold.dev

.PHONY: preview-docs-v2
preview-docs-v2:
	./deploy/docs-v2/local-preview.sh hugo serve -D --bind=0.0.0.0 --ignoreCache

.PHONY: build-docs-v2-preview
build-docs-preview-v2:
	./deploy/docs-v2/local-preview.sh hugo --baseURL=https://skaffold-v2.web.app

# schema generation

.PHONY: generate-schemas
generate-schemas:
	go run hack/schemas/main.go

.PHONY: generate-schemas-v2
generate-schemas-v2:
	go run hack/schemas/main.go

# telemetry generation
.PHONY: generate-schemas
generate-telemetry-json:
	go run hack/struct-json/main.go -- pkg/skaffold/instrumentation/types.go docs-v1/content/en/docs/resources/telemetry/metrics.json

# telemetry generation
.PHONY: generate-schemas-v2
generate-telemetry-json-v2:
	go run hack/struct-json/main.go -- pkg/skaffold/instrumentation/types.go docs-v2/content/en/docs/resources/telemetry/metrics.json

# dashboards
.PHONY: generate-flags-dashboard
flags-dashboard:
	go run hack/patch-dashboard/main.go $(GCP_PROJECT)

# static files

$(EMBEDDED_FILES_CHECK): go.mod docs-v1/content/en/schemas/* docs-v2/content/en/schemas/*
	hack/generate-embedded-files.sh

# run comparisonstats - ex: make COMPARISONSTATS_ARGS='usr/local/bin/skaffold /usr/local/bin/skaffold helm-deployment main.go "//per-dev-iteration-comment"' comparisonstats
.PHONY: comparisonstats
comparisonstats:
	go run hack/comparisonstats/main.go $(COMPARISONSTATS_ARGS)
