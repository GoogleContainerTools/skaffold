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
GOARCH = amd64
BUILD_DIR ?= ./out
ORG := github.com/GoogleContainerTools
PROJECT := skaffold
REPOPATH ?= $(ORG)/$(PROJECT)
RELEASE_BUCKET ?= $(PROJECT)
GSC_BUILD_PATH ?= gs://$(RELEASE_BUCKET)/builds/$(COMMIT)
GSC_BUILD_LATEST ?= gs://$(RELEASE_BUCKET)/builds/latest
GSC_RELEASE_PATH ?= gs://$(RELEASE_BUCKET)/releases/$(VERSION)
GSC_RELEASE_LATEST ?= gs://$(RELEASE_BUCKET)/releases/latest

GCP_ONLY ?= false
GCP_PROJECT ?= k8s-skaffold
GKE_CLUSTER_NAME ?= integration-tests
GKE_ZONE ?= us-central1-a

SUPPORTED_PLATFORMS := linux-$(GOARCH) darwin-$(GOARCH) windows-$(GOARCH).exe
BUILD_PACKAGE = $(REPOPATH)/cmd/skaffold

VERSION_PACKAGE = $(REPOPATH)/pkg/skaffold/version
COMMIT = $(shell git rev-parse HEAD)

ifeq "$(strip $(VERSION))" ""
 override VERSION = $(shell git describe --always --tags --dirty)
endif

# Force using Go Modules and always read the dependencies from
# the `vendor` folder.
export GO111MODULE = on
export GOFLAGS = -mod=vendor

GO_GCFLAGS := "all=-trimpath=${PWD}"
GO_ASMFLAGS := "all=-trimpath=${PWD}"

LDFLAGS_linux = -static
LDFLAGS_darwin =
LDFLAGS_windows =

GO_BUILD_TAGS_linux := "osusergo netgo static_build"
GO_BUILD_TAGS_darwin := ""
GO_BUILD_TAGS_windows := ""

GO_LDFLAGS = -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -X $(VERSION_PACKAGE).buildDate=$(shell date +'%Y-%m-%dT%H:%M:%SZ')
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitCommit=$(COMMIT)
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitTreeState=$(if $(shell git status --porcelain),dirty,clean)

GO_LDFLAGS_windows =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_windows)\""
GO_LDFLAGS_darwin =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_darwin)\""
GO_LDFLAGS_linux =" $(GO_LDFLAGS)  -extldflags \"$(LDFLAGS_linux)\""

GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

$(BUILD_DIR)/$(PROJECT): $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH) $@

$(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH): $(GO_FILES) $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -tags $(GO_BUILD_TAGS_$(GOOS)) -ldflags $(GO_LDFLAGS_$(GOOS)) -gcflags $(GO_GCFLAGS) -asmflags $(GO_ASMFLAGS) -o $@ $(BUILD_PACKAGE)

$(BUILD_DIR)/$(PROJECT)-%-$(GOARCH): $(GO_FILES) $(BUILD_DIR)
	docker build --build-arg PROJECT=$(REPOPATH) \
		--build-arg TARGETS=$*/$(GOARCH) \
		--build-arg FLAG_LDFLAGS=$(GO_LDFLAGS_$(*)) \
		--build-arg FLAG_TAGS=$(GO_BUILD_TAGS_$(*)) \
		-f deploy/cross/Dockerfile -t skaffold/cross .
	docker run --rm --entrypoint sh skaffold/cross -c "cat /build/skaffold*" > $@

%.sha256: %
	shasum -a 256 $< > $@

%.exe: %
	cp $< $@

.PHONY: $(BUILD_DIR)/VERSION
$(BUILD_DIR)/VERSION: $(BUILD_DIR)
	@ echo $(VERSION) > $@

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PRECIOUS: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

.PHONY: cross
cross: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform).sha256)

.PHONY: test
test: $(BUILD_DIR)
	@ ./hack/test.sh
	@ ./hack/checks.sh

.PHONY: coverage
coverage: $(BUILD_DIR)
	@ ./hack/test.sh
	@ curl -s https://codecov.io/bash > $(BUILD_DIR)/upload_coverage
	@ bash $(BUILD_DIR)/upload_coverage

.PHONY: checks
checks: $(BUILD_DIR)
	@ ./hack/checks.sh

.PHONY: quicktest
quicktest:
	go test -short -timeout=60s ./...

.PHONY: install
install: $(GO_FILES) $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go install -tags $(GO_BUILD_TAGS_$(GOOS)) -ldflags $(GO_LDFLAGS_$(GOOS)) -gcflags $(GO_GCFLAGS) -asmflags $(GO_ASMFLAGS) $(BUILD_PACKAGE)

.PHONY: integration
integration: install
ifeq ($(GCP_ONLY),true)
	gcloud container clusters get-credentials \
		$(GKE_CLUSTER_NAME) \
		--zone $(GKE_ZONE) \
		--project $(GCP_PROJECT)
endif
#	kubectl get nodes -oyaml
	GCP_ONLY=$(GCP_ONLY) go test -v $(REPOPATH)/integration -timeout 20m $(INTEGRATION_TEST_ARGS)

.PHONY: release
release: cross $(BUILD_DIR)/VERSION
	docker build \
		-f deploy/skaffold/Dockerfile \
		--cache-from gcr.io/$(GCP_PROJECT)/skaffold-builder \
		--build-arg VERSION=$(VERSION) \
		-t gcr.io/$(GCP_PROJECT)/skaffold:latest \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(VERSION) .
	gsutil -m cp $(BUILD_DIR)/$(PROJECT)-* $(GSC_RELEASE_PATH)/
	gsutil -m cp $(BUILD_DIR)/VERSION $(GSC_RELEASE_PATH)/VERSION
	gsutil -m cp -r $(GSC_RELEASE_PATH)/* $(GSC_RELEASE_LATEST)

.PHONY: release-in-docker
release-in-docker:
	docker build \
		-f deploy/skaffold/Dockerfile \
		-t gcr.io/$(GCP_PROJECT)/skaffold-builder \
		--target builder \
		.
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.config/gcloud:/root/.config/gcloud \
		gcr.io/$(GCP_PROJECT)/skaffold-builder make -j release VERSION=$(VERSION) RELEASE_BUCKET=$(RELEASE_BUCKET) GCP_PROJECT=$(GCP_PROJECT)

.PHONY: release-build
release-build: cross
	docker build \
		-f deploy/skaffold/Dockerfile \
		--cache-from gcr.io/$(GCP_PROJECT)/skaffold-builder \
		-t gcr.io/$(GCP_PROJECT)/skaffold:edge \
		-t gcr.io/$(GCP_PROJECT)/skaffold:$(COMMIT) .
	gsutil -m cp $(BUILD_DIR)/$(PROJECT)-* $(GSC_BUILD_PATH)/
	gsutil -m cp -r $(GSC_BUILD_PATH)/* $(GSC_BUILD_LATEST)

.PHONY: release-build-in-docker
release-build-in-docker:
	docker build \
		-f deploy/skaffold/Dockerfile \
		-t gcr.io/$(GCP_PROJECT)/skaffold-builder \
		--target builder \
		.
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.config/gcloud:/root/.config/gcloud \
		gcr.io/$(GCP_PROJECT)/skaffold-builder make -j release-build RELEASE_BUCKET=$(RELEASE_BUCKET) GCP_PROJECT=$(GCP_PROJECT)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

.PHONY: kind-cluster
kind-cluster:
	kind get clusters | grep -q kind || kind create cluster --image=kindest/node:v1.13.10@sha256:2f5f882a6d0527a2284d29042f3a6a07402e1699d792d0d5a9b9a48ef155fa2a

.PHONY: skaffold-builder
skaffold-builder:
	-docker pull gcr.io/$(GCP_PROJECT)/skaffold-builder
	docker build \
		--cache-from gcr.io/$(GCP_PROJECT)/skaffold-builder \
		-f deploy/skaffold/Dockerfile \
		--target integration \
		-t gcr.io/$(GCP_PROJECT)/skaffold-integration .

.PHONY: integration-in-kind
integration-in-kind: kind-cluster skaffold-builder
	docker exec -it kind-control-plane cat /etc/kubernetes/admin.conf > /tmp/kind-config
	echo '{}' > /tmp/docker-config
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v /tmp/kind-config:/kind-config \
		-v /tmp/docker-config:/root/.docker/config.json \
		-e KUBECONFIG=/kind-config \
		gcr.io/$(GCP_PROJECT)/skaffold-integration

.PHONY: integration-in-docker
integration-in-docker: skaffold-builder
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.config/gcloud:/root/.config/gcloud \
		-v $(GOOGLE_APPLICATION_CREDENTIALS):$(GOOGLE_APPLICATION_CREDENTIALS) \
		-e GCP_ONLY=$(GCP_ONLY) \
		-e GCP_PROJECT=$(GCP_PROJECT) \
		-e GKE_CLUSTER_NAME=$(GKE_CLUSTER_NAME) \
		-e GKE_ZONE=$(GKE_ZONE) \
		-e DOCKER_CONFIG=/root/.docker \
		-e GOOGLE_APPLICATION_CREDENTIALS=$(GOOGLE_APPLICATION_CREDENTIALS) \
		-e INTEGRATION_TEST_ARGS=$(INTEGRATION_TEST_ARGS) \
		gcr.io/$(GCP_PROJECT)/skaffold-integration

.PHONY: submit-build-trigger
submit-build-trigger:
	gcloud container builds submit . \
		--config=deploy/cloudbuild.yaml \
		--substitutions="_RELEASE_BUCKET=$(RELEASE_BUCKET),COMMIT_SHA=$(COMMIT)"

.PHONY: submit-release-trigger
submit-release-trigger:
	gcloud container builds submit . \
		--config=deploy/cloudbuild-release.yaml \
		--substitutions="_RELEASE_BUCKET=$(RELEASE_BUCKET),TAG_NAME=$(VERSION)"

#utilities for skaffold site - not used anywhere else

.PHONY: preview-docs
preview-docs:
	./deploy/docs/local-preview.sh hugo serve -D --bind=0.0.0.0

.PHONY: build-docs-preview
build-docs-preview:
	./deploy/docs/local-preview.sh hugo --baseURL=https://skaffold.dev

# schema generation

.PHONY: generate-schemas
generate-schemas:
	go run hack/schemas/main.go
