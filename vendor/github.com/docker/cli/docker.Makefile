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


# build executable using a container
binary: build_binary_native_image
	docker run --rm $(ENVVARS) $(MOUNTS) $(BINARY_NATIVE_IMAGE_NAME)

build: binary


# clean build artifacts using a container
.PHONY: clean
clean: build_docker_image
	docker run --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make clean

# run go test
.PHONY: test-unit
test-unit: build_docker_image
	docker run --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make test-unit

.PHONY: test
test: test-unit test-e2e

# build the CLI for multiple architectures using a container
.PHONY: cross
cross: build_cross_image
	docker run --rm $(ENVVARS) $(MOUNTS) $(CROSS_IMAGE_NAME) make cross

.PHONY: binary-windows
binary-windows: build_cross_image
	docker run --rm $(ENVVARS) $(MOUNTS) $(CROSS_IMAGE_NAME) make $@

.PHONY: binary-osx
binary-osx: build_cross_image
	docker run --rm $(ENVVARS) $(MOUNTS) $(CROSS_IMAGE_NAME) make $@

.PHONY: watch
watch: build_docker_image
	docker run --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make watch

# start container in interactive mode for in-container development
.PHONY: dev
dev: build_docker_image
	docker run -ti $(ENVVARS) $(MOUNTS) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		$(DEV_DOCKER_IMAGE_NAME) ash

shell: dev

# run linters in a container
.PHONY: lint
lint: build_linter_image
	docker run -ti $(ENVVARS) $(MOUNTS) $(LINTER_IMAGE_NAME)

# download dependencies (vendor/) listed in vendor.conf, using a container
.PHONY: vendor
vendor: build_docker_image vendor.conf
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make vendor

dynbinary: build_cross_image
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(CROSS_IMAGE_NAME) make dynbinary

.PHONY: authors
authors: ## generate AUTHORS file from git history
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make authors

## generate man pages from go source and markdown
.PHONY: manpages
manpages: build_docker_image
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make manpages

## Generate documentation YAML files consumed by docs repo
.PHONY: yamldocs
yamldocs: build_docker_image
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(DEV_DOCKER_IMAGE_NAME) make yamldocs

.PHONY: shellcheck
shellcheck: build_shell_validate_image
	docker run -ti --rm $(ENVVARS) $(MOUNTS) $(VALIDATE_IMAGE_NAME) make shellcheck

.PHONY: test-e2e
test-e2e: binary
	./scripts/test/e2e/wrapper
