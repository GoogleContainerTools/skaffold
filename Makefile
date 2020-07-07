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
GOOS ?= $(shell go env GOOS)
GOARCH ?= amd64
BUILD_DIR ?= ./out
ORG = github.com/GoogleContainerTools
PROJECT = skaffold
REPOPATH ?= $(ORG)/$(PROJECT)
RELEASE_BUCKET ?= $(PROJECT)
GSC_BUILD_PATH ?= gs://$(RELEASE_BUCKET)/builds/$(COMMIT)
GSC_BUILD_LATEST ?= gs://$(RELEASE_BUCKET)/builds/latest
GSC_RELEASE_PATH ?= gs://$(RELEASE_BUCKET)/releases/$(VERSION)
GSC_RELEASE_LATEST ?= gs://$(RELEASE_BUCKET)/releases/latest
KIND_NODE ?= kindest/node:v1.13.12@sha256:214476f1514e47fe3f6f54d0f9e24cfb1e4cda449529791286c7161b7f9c08e7

GCP_ONLY ?= false
GCP_PROJECT ?= k8s-skaffold
GKE_CLUSTER_NAME ?= integration-tests
GKE_ZONE ?= us-central1-a

SUPPORTED_PLATFORMS = linux-amd64 darwin-amd64 windows-amd64.exe linux-arm64
BUILD_PACKAGE = $(REPOPATH)/cmd/skaffold

SKAFFOLD_TEST_PACKAGES = ./pkg/skaffold/... ./cmd/... ./hack/... ./pkg/webhook/...
GO_FILES = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./pkg/diag/*")

VERSION_PACKAGE = $(REPOPATH)/pkg/skaffold/version
COMMIT = $(shell git rev-parse HEAD)

ifeq "$(strip $(VERSION))" ""
 override VERSION = $(shell git describe --always --tags --dirty)
endif

LDFLAGS_linux = -static
LDFLAGS_darwin =
LDFLAGS_windows =

GO_BUILD_TAGS_linux = "osusergo netgo static_build release"
GO_BUILD_TAGS_darwin = "release"
GO_BUILD_TAGS_windows = "release"

GO_LDFLAGS = -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -X $(VERSION_PACKAGE).buildDate=$(shell date +'%Y-%m-%dT%H:%M:%SZ')
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitCommit=$(COMMIT)
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitTreeState=$(if $(shell git status --porcelain),dirty,clean)
GO_LDFLAGS += -s -w

GO_LDFLAGS_windows =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_windows)\""
GO_LDFLAGS_darwin =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_darwin)\""
GO_LDFLAGS_linux =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_linux)\""

STATIK_FILES = cmd/skaffold/app/cmd/statik/statik.go

# Build for local development.
$(BUILD_DIR)/$(PROJECT): $(STATIK_FILES) $(GO_FILES) $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -tags $(GO_BUILD_TAGS_$(GOOS)) -ldflags $(GO_LDFLAGS_$(GOOS)) -o $@ $(BUILD_PACKAGE)

.PHONY: install
install: $(BUILD_DIR)/$(PROJECT)
	cp $(BUILD_DIR)/$(PROJECT) $(GOPATH)/bin/$(PROJECT)

.PRECIOUS: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

.PHONY: cross
cross: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

$(BUILD_DIR)/$(PROJECT)-%: $(STATIK_FILES) $(GO_FILES) $(BUILD_DIR) deploy/cross/Dockerfile
	$(eval os = $(firstword $(subst -, ,$*)))
	$(eval arch = $(lastword $(subst -, ,$(subst .exe,,$*))))
	$(eval ldflags = $(GO_LDFLAGS_$(os)))
	$(eval tags = $(GO_BUILD_TAGS_$(os)))

	docker build \
		--build-arg GOOS=$(os) \
		--build-arg GOARCH=$(arch) \
		--build-arg TAGS=$(tags) \
		--build-arg LDFLAGS=$(ldflags) \
		-f deploy/cross/Dockerfile \
		-t skaffold/cross \
		.

	docker run --rm skaffold/cross cat /build/skaffold > $@
	shasum -a 256 $@ | tee $@.sha256
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

.PHONY: coverage
coverage: $(BUILD_DIR)
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
endif
	@ GCP_ONLY=$(GCP_ONLY) ./hack/gotest.sh -v $(REPOPATH)/integration/binpack $(REPOPATH)/integration -timeout 20m $(INTEGRATION_TEST_ARGS)

.PHONY: integration
integration: install integration-tests

.PHONY: release
release: cross $(BUILD_DIR)/VERSION
	docker build \
		--build-arg VERSION=$(VERSION) \
		-f deploy/skaffold/Dockerfile \
		--target release \
		-t gcr.io/$(GCP_PROJECT)/skaffold:latest \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(VERSION) \
		.
	gsutil -m cp $(BUILD_DIR)/$(PROJECT)-* $(GSC_RELEASE_PATH)/
	gsutil -m cp $(BUILD_DIR)/VERSION $(GSC_RELEASE_PATH)/VERSION
	gsutil -m cp -r $(GSC_RELEASE_PATH)/* $(GSC_RELEASE_LATEST)

.PHONY: release-build
release-build: cross
	docker build \
		-f deploy/skaffold/Dockerfile \
		--target release \
		-t gcr.io/$(GCP_PROJECT)/skaffold:edge \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(COMMIT) \
		.
	gsutil -m cp $(BUILD_DIR)/$(PROJECT)-* $(GSC_BUILD_PATH)/
	gsutil -m cp -r $(GSC_BUILD_PATH)/* $(GSC_BUILD_LATEST)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) hack/bin $(STATIK_FILES)

.PHONY: build_deps
build_deps:
	$(eval DEPS_DIGEST := $(shell ./hack/skaffold-deps-sha1.sh))
	docker build \
		-f deploy/skaffold/Dockerfile.deps \
		-t gcr.io/$(GCP_PROJECT)/build_deps:$(DEPS_DIGEST) \
		deploy/skaffold
	docker push gcr.io/$(GCP_PROJECT)/build_deps:$(DEPS_DIGEST)
	@./hack/check-skaffold-builder.sh

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
	docker pull $(KIND_NODE)
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
			kind get clusters | grep -q kind || TERM=dumb kind create cluster --image=$(KIND_NODE); \
			kind get kubeconfig --internal > /tmp/kind-config; \
			make integration \
		'

.PHONY: integration-in-docker
integration-in-docker: skaffold-builder
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
		gcr.io/$(GCP_PROJECT)/skaffold-builder \
		make integration

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
	./deploy/docs/local-preview.sh hugo serve -D --bind=0.0.0.0 --ignoreCache

.PHONY: build-docs-preview
build-docs-preview:
	./deploy/docs/local-preview.sh hugo --baseURL=https://skaffold.dev

# schema generation

.PHONY: generate-schemas
generate-schemas:
	go run hack/schemas/main.go

# static files

$(STATIK_FILES): go.mod docs/content/en/schemas/*
	hack/generate-statik.sh
