# Copyright 2018 The Skaffold Authors
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

REMOTE_INTEGRATION ?= false
GCP_PROJECT ?= k8s-skaffold
GKE_CLUSTER_NAME ?= integration-tests
GKE_ZONE ?= us-central1-a

SUPPORTED_PLATFORMS := linux-$(GOARCH) darwin-$(GOARCH) windows-$(GOARCH).exe
BUILD_PACKAGE = $(REPOPATH)/cmd/skaffold

VERSION_PACKAGE = $(REPOPATH)/pkg/skaffold/version
COMMIT = $(shell git rev-parse HEAD)
BASE_URL ?= https://skaffold.dev
VERSION ?= $(shell git describe --always --tags --dirty)

GO_GCFLAGS := "all=-trimpath=${PWD}"
GO_ASMFLAGS := "all=-trimpath=${PWD}"

GO_LDFLAGS :="
GO_LDFLAGS += -extldflags \"${LDFLAGS}\"
GO_LDFLAGS += -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -X $(VERSION_PACKAGE).buildDate=$(shell date +'%Y-%m-%dT%H:%M:%SZ')
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitCommit=$(COMMIT)
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitTreeState=$(if $(shell git status --porcelain),dirty,clean)
GO_LDFLAGS +="

GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GO_BUILD_TAGS := "kqueue"

DOCSY_COMMIT:=a7141a2eac26cb598b707cab87d224f9105c315d

$(BUILD_DIR)/$(PROJECT): $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH) $@

$(BUILD_DIR)/$(PROJECT)-%-$(GOARCH): $(GO_FILES) $(BUILD_DIR)
	GOOS=$* GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags $(GO_LDFLAGS) -gcflags $(GO_GCFLAGS) -asmflags $(GO_ASMFLAGS) -tags $(GO_BUILD_TAGS) -o $@ $(BUILD_PACKAGE)

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
test:
	@ ./test.sh

.PHONY: install
install: $(GO_FILES) $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go install -ldflags $(GO_LDFLAGS) -gcflags $(GO_GCFLAGS) -asmflags $(GO_ASMFLAGS) -tags $(GO_BUILD_TAGS) $(BUILD_PACKAGE)

.PHONY: integration
integration: install $(BUILD_DIR)/$(PROJECT)
	go test -v -tags integration $(REPOPATH)/integration -timeout 10m --remote=$(REMOTE_INTEGRATION)

.PHONY: coverage
coverage: $(BUILD_DIR)
	go test -coverprofile=$(BUILD_DIR)/coverage.txt -covermode=atomic ./...

.PHONY: release
release: cross $(BUILD_DIR)/VERSION
	docker build \
        		-f deploy/skaffold/Dockerfile \
        		--cache-from gcr.io/$(GCP_PROJECT)/skaffold-builder \
        		--build-arg VERSION=$(VERSION) \
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
	docker run \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.config/gcloud:/root/.config/gcloud \
		gcr.io/$(GCP_PROJECT)/skaffold-builder make -j release VERSION=$(VERSION) RELEASE_BUCKET=$(RELEASE_BUCKET) GCP_PROJECT=$(GCP_PROJECT)

.PHONY: release-build
release-build: cross
	docker build \
    		-f deploy/skaffold/Dockerfile \
    		--cache-from gcr.io/$(GCP_PROJECT)/skaffold-builder \
    		-t gcr.io/$(GCP_PROJECT)/skaffold:latest \
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
	docker run \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.config/gcloud:/root/.config/gcloud \
		gcr.io/$(GCP_PROJECT)/skaffold-builder make -j release-build RELEASE_BUCKET=$(RELEASE_BUCKET) GCP_PROJECT=$(GCP_PROJECT)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) ./docs/public ./docs/resources

.PHONY: integration-in-docker
integration-in-docker:
	docker build \
		-f deploy/skaffold/Dockerfile \
		--target integration \
		-t gcr.io/$(GCP_PROJECT)/skaffold-integration .
	docker run \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(HOME)/.config/gcloud:/root/.config/gcloud \
		-v $(GOOGLE_APPLICATION_CREDENTIALS):$(GOOGLE_APPLICATION_CREDENTIALS) \
		-e REMOTE_INTEGRATION=true \
		-e DOCKER_CONFIG=/root/.docker \
		-e GOOGLE_APPLICATION_CREDENTIALS=$(GOOGLE_APPLICATION_CREDENTIALS) \
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

.PHONY: docs-controller-image
docs-controller-image: 
	docker build -t gcr.io/$(GCP_PROJECT)/docs-controller -f deploy/webhook/Dockerfile .


.PHONY: preview-docs
preview-docs: start-docs-preview clean-docs-preview

.PHONY: docs-preview-image
docs-preview-image:
	docker build -t skaffold-docs-previewer -f deploy/webhook/Dockerfile --target runtime_deps .

.PHONY: start-docs-preview
start-docs-preview:	docs-preview-image
	docker run -ti -v $(PWD):/app --workdir /app/ -p 1313:1313 skaffold-docs-previewer bash -xc deploy/docs/preview.sh

.PHONY: build-docs-preview
build-docs-preview:	docs-preview-image
	docker run -ti -v $(PWD):/app --workdir /app/ -p 1313:1313 skaffold-docs-previewer bash -xc deploy/docs/build.sh

.PHONY: clean-docs-preview
clean-docs-preview: docs-preview-image
	docker run -ti -v $(PWD):/app --workdir /app/ -p 1313:1313 skaffold-docs-previewer bash -xc deploy/docs/clean.sh
