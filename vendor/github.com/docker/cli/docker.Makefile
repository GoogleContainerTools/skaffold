#
# github.com/docker/cli
#
# Makefile for developing using Docker
#

DEV_DOCKER_IMAGE_NAME = docker-cli-dev$(IMAGE_TAG)
BINARY_NATIVE_IMAGE_NAME = docker-cli-native$(IMAGE_TAG)
LINTER_IMAGE_NAME = docker-cli-lint$(IMAGE_TAG)
CROSS_IMAGE_NAME = docker-cli-cross$(IMAGE_TAG)
VALIDATE_IMAGE_NAME = docker-cli-shell-validate$(IMAGE_TAG)
E2E_IMAGE_NAME = docker-cli-e2e$(IMAGE_TAG)
MOUNTS = -v "$(CURDIR)":/go/src/github.com/docker/cli
VERSION = $(shell cat VERSION)
ENVVARS = -e VERSION=$(VERSION) -e GITCOMMIT -e PLATFORM

# build docker image (dockerfiles/Dockerfile.build)
.PHONY: build_docker_image
build_docker_image:
	docker build ${DOCKER_BUILD_ARGS} -t $(DEV_DOCKER_IMAGE_NAME) -f ./dockerfiles/Dockerfile.dev .

# build docker image having the linting tools (dockerfiles/Dockerfile.lint)
.PHONY: build_linter_image
build_linter_image:
	docker build ${DOCKER_BUILD_ARGS} -t $(LINTER_IMAGE_NAME) -f ./dockerfiles/Dockerfile.lint .

.PHONY: build_cross_image
build_cross_image:
	docker build ${DOCKER_BUILD_ARGS} -t $(CROSS_IMAGE_NAME) -f ./dockerfiles/Dockerfile.cross .

.PHONY: build_shell_validate_image
build_shell_validate_image:
	docker build -t $(VALIDATE_IMAGE_NAME) -f ./dockerfiles/Dockerfile.shellcheck .

.PHONY: build_binary_native_image
build_binary_native_image:
	docker build -t $(BINARY_NATIVE_IMAGE_NAME) -f ./dockerfiles/Dockerfile.binary-native .

.PHONY: build_e2e_image
build_e2e_image:
	docker build -t $(E2E_IMAGE_NAME) --build-arg VERSION=$(VERSION) --build-arg GITCOMMIT=$(GITCOMMIT) -f ./dockerfiles/Dockerfile.e2e .


binary: build_binary_native_image ## build the CLI
	docker run --rm $(ENVVARS) $(MOUNTS) $(BINARY_NATIVE_IMAGE_NAME)

build: binary ## alias for binary

.PHONY: clean
clean: build_docker_image ## clean build artifacts
	docker run --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make clean

.PHONY: test-unit
test-unit: build_docker_image # run unit tests (using go test)
	docker run --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make test-unit

.PHONY: test ## run unit and e2e tests
test: test-unit test-e2e

.PHONY: cross
cross: build_cross_image ## build the CLI for macOS and Windows
	docker run --rm $(ENVVARS) $(MOUNTS) $(CROSS_IMAGE_NAME) make cross

.PHONY: binary-windows
binary-windows: build_cross_image ## build the CLI for Windows
	docker run --rm $(ENVVARS) $(MOUNTS) $(CROSS_IMAGE_NAME) make $@

.PHONY: binary-osx
binary-osx: build_cross_image ## build the CLI for macOS
	docker run --rm $(ENVVARS) $(MOUNTS) $(CROSS_IMAGE_NAME) make $@

.PHONY: dev
dev: build_docker_image ## start a build container in interactive mode for in-container development
	docker run -ti $(ENVVARS) $(MOUNTS) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		$(DEV_DOCKER_IMAGE_NAME) ash

shell: dev ## alias for dev

.PHONY: lint
lint: build_linter_image ## run linters
	docker run -ti $(ENVVARS) $(MOUNTS) $(LINTER_IMAGE_NAME)

.PHONY: vendor
vendor: build_docker_image vendor.conf ## download dependencies (vendor/) listed in vendor.conf
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make vendor

dynbinary: build_cross_image ## build the CLI dynamically linked
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(CROSS_IMAGE_NAME) make dynbinary

.PHONY: authors
authors: ## generate AUTHORS file from git history
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make authors

.PHONY: manpages
manpages: build_docker_image ## generate man pages from go source and markdown
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make manpages

.PHONY: yamldocs
yamldocs: build_docker_image ## generate documentation YAML files consumed by docs repo
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make yamldocs

.PHONY: shellcheck
shellcheck: build_shell_validate_image ## run shellcheck validation
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(VALIDATE_IMAGE_NAME) make shellcheck

.PHONY: test-e2e ## run e2e tests
test-e2e: test-e2e-non-experimental test-e2e-experimental

.PHONY: test-e2e-experimental
test-e2e-experimental: build_e2e_image
	docker run -e DOCKERD_EXPERIMENTAL=1 --rm -v /var/run/docker.sock:/var/run/docker.sock $(E2E_IMAGE_NAME)

.PHONY: test-e2e-non-experimental
test-e2e-non-experimental: build_e2e_image
	docker run --rm -v /var/run/docker.sock:/var/run/docker.sock $(E2E_IMAGE_NAME)

.PHONY: help
help: ## print this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
