# Copyright 2017 The Kubernetes Authors. All rights reserved.
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

TAG = v6
REGISTRY = gcr.io/yuwenma-gke-playground
ARCH ?= $(shell go env GOARCH)
ALL_ARCH = amd64 arm arm64 ppc64le

QEMUVERSION=v2.7.0

IMAGE = $(REGISTRY)/gb-frontend
MULTI_ARCH_IMG = $(IMAGE)-$(ARCH)

# Set default base image dynamically for each arch
ifeq ($(ARCH),arm)
	QEMUARCH=arm
	BASEIMAGE=armhf/php:5.6-apache
endif
ifeq ($(ARCH),arm64)
	QEMUARCH=aarch64
	BASEIMAGE=arm64v8/php:5-apache
endif
ifeq ($(ARCH),ppc64le)
	QEMUARCH=ppc64le
	BASEIMAGE=ppc64le/php:5-apache
endif

TEMP_DIR := $(shell mktemp -d)

all: all-container

sub-container-%:
	$(MAKE) ARCH=$* container

sub-push-%:
	$(MAKE) ARCH=$* push

all-container: $(addprefix sub-container-,$(ALL_ARCH))

all-push: $(addprefix sub-push-,$(ALL_ARCH))
	docker manifest create --amend $(IMAGE):$(TAG) $(shell echo $(ALL_ARCH) | sed -e "s~[^ ]*~$(IMAGE)\-&:$(TAG)~g")
	@for arch in $(ALL_ARCH); do docker manifest annotate --arch $${arch} ${IMAGE}:${TAG} ${IMAGE}-$${arch}:${TAG}; done
	docker manifest push --purge ${IMAGE}:${TAG}

container: .container-$(ARCH)
.container-$(ARCH):
	cp ./* $(TEMP_DIR)

ifneq ($(ARCH),amd64)
	cd $(TEMP_DIR) && sed -i "s|FROM php.*|FROM $(BASEIMAGE)\nCOPY qemu-$(QEMUARCH)-static /usr/bin\n|g" Dockerfile
	# Register /usr/bin/qemu-ARCH-static as the handler for ARM binaries in the kernel
	docker run --rm --privileged multiarch/qemu-user-static:register --reset
	curl -sSL https://github.com/multiarch/qemu-user-static/releases/download/$(QEMUVERSION)/x86_64_qemu-$(QEMUARCH)-static.tar.gz | tar -xz -C $(TEMP_DIR)
endif
	docker build -t $(MULTI_ARCH_IMG):$(TAG) $(TEMP_DIR)

ifeq ($(ARCH), amd64)
	# This is for to maintain the backward compatibility
	docker tag $(MULTI_ARCH_IMG):$(TAG) $(IMAGE):$(TAG)
endif

push: .push-$(ARCH)
.push-$(ARCH): .container-$(ARCH)
	docker push $(MULTI_ARCH_IMG):$(TAG)

clean: $(addprefix sub-clean-,$(ALL_ARCH))
	docker rmi -f $(IMAGE):$(TAG) || true
sub-clean-%:
	docker rmi -f $(IMAGE)-$*:$(TAG) || true
