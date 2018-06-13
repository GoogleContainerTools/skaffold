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

# Bump these on release
VERSION_MAJOR ?= 0
VERSION_MINOR ?= 7
VERSION_BUILD ?= 0

VERSION ?= v$(VERSION_MAJOR).$(VERSION_MINOR).$(VERSION_BUILD)

GOOS ?= $(shell go env GOOS)
GOARCH = amd64
BUILD_DIR ?= ./out
DOCS_DIR ?= ./docs/generated
ORG := github.com/GoogleContainerTools
PROJECT := skaffold
REPOPATH ?= $(ORG)/$(PROJECT)
RELEASE_BUCKET ?= $(PROJECT)

REMOTE_INTEGRATION ?= false
GCP_PROJECT ?= k8s-skaffold
GKE_CLUSTER_NAME ?= integration-tests
GKE_ZONE ?= us-central1-a

SUPPORTED_PLATFORMS := linux-$(GOARCH) darwin-$(GOARCH) windows-$(GOARCH).exe
BUILD_PACKAGE = $(REPOPATH)/cmd/skaffold

VERSION_PACKAGE = $(REPOPATH)/pkg/skaffold/version
COMMIT = $(shell git rev-parse HEAD)

GO_LDFLAGS :="
GO_LDFLAGS += -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -X $(VERSION_PACKAGE).buildDate=$(shell date +'%Y-%m-%dT%H:%M:%SZ')
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitCommit=$(COMMIT)
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitTreeState=$(if $(shell git status --porcelain),dirty,clean)
GO_LDFLAGS +="

GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
GO_BUILD_TAGS := "kqueue"

$(BUILD_DIR)/$(PROJECT): $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH)
	cp $(BUILD_DIR)/$(PROJECT)-$(GOOS)-$(GOARCH) $@

$(BUILD_DIR)/$(PROJECT)-%-$(GOARCH): $(GO_FILES) $(BUILD_DIR)
	GOOS=$* GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags $(GO_LDFLAGS) -tags $(GO_BUILD_TAGS) -o $@ $(BUILD_PACKAGE)

%.sha256: %
	shasum -a 256 $< &> $@

%.exe: %
	mv $< $@

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
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go install -ldflags $(GO_LDFLAGS) -tags $(GO_BUILD_TAGS) $(BUILD_PACKAGE)

.PHONY: integration
integration: install $(BUILD_DIR)/$(PROJECT)
	go test -v -tags integration $(REPOPATH)/integration -timeout 10m --remote=$(REMOTE_INTEGRATION)

.PHONY: release
release: cross docs
	docker build \
        		-f deploy/skaffold/Dockerfile \
        		--cache-from gcr.io/$(GCP_PROJECT)/skaffold-builder \
        		-t gcr.io/$(GCP_PROJECT)/skaffold:$(VERSION) .
	gsutil -m cp  gs://$(RELEASE_BUCKET)/releases/$(VERSION)/
	gsutil -m cp $(DOCS_DIR)/* gs://$(RELEASE_BUCKET)/releases/$(VERSION)/docs/
	gsutil -m cp $(BUILD_DIR)/$(PROJECT)-* gs://$(RELEASE_BUCKET)/latest/

.PHONY: release-build
release-build: cross docs
	docker build \
    		-f deploy/skaffold/Dockerfile \
    		--cache-from gcr.io/$(GCP_PROJECT)/skaffold-builder \
    		-t gcr.io/$(GCP_PROJECT)/skaffold:$(COMMIT) .
	gsutil -m cp $(BUILD_DIR)/$(PROJECT)-* gs://$(RELEASE_BUCKET)/builds/$(COMMIT)/
	gsutil -m cp $(DOCS_DIR)/* gs://$(RELEASE_BUCKET)/releases/$(COMMIT)/docs/
	gsutil -m cp $(BUILD_DIR)/$(PROJECT)-* gs://$(RELEASE_BUCKET)/builds/latest/

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
		gcr.io/$(GCP_PROJECT)/skaffold-builder make release-build RELEASE_BUCKET=$(RELEASE_BUCKET) GCP_PROJECT=$(GCP_PROJECT)

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
		gcr.io/$(GCP_PROJECT)/skaffold-builder make release RELEASE_BUCKET=$(RELEASE_BUCKET) GCP_PROJECT=$(GCP_PROJECT)

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

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

.PHONY: docs
docs:
	hack/build_docs.sh $(VERSION) $(COMMIT)

.PHONY: docs-in-docker
docs-in-docker:
	docker build \
		-f deploy/skaffold/Dockerfile \
		-t skaffold-builder \
		--target builder \
		.
	docker run \
		-v $(PWD):/go/src/$(REPOPATH) \
		skaffold-builder make docs

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

