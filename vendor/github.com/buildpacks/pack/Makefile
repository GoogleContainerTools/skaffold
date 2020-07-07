ACCEPTANCE_TIMEOUT?=$(TEST_TIMEOUT)
ARCHIVE_NAME=pack-$(PACK_VERSION)
GOCMD?=go
GOFLAGS?=-mod=vendor
GOTESTFLAGS?=-v -count=1 -parallel=1
PACKAGE_BASE=github.com/buildpacks/pack
PACK_BIN?=pack
PACK_GITSHA1=$(shell git rev-parse --short=7 HEAD)
PACK_VERSION?=0.0.0
SRC=$(shell find . -type f -name '*.go' -not -path "*/vendor/*")
TEST_TIMEOUT?=900s
UNIT_TIMEOUT?=$(TEST_TIMEOUT)

clean_build := $(strip ${PACK_BUILD})
clean_sha := $(strip ${PACK_GITSHA1})

# append build number and git sha to version, if not-empty
ifneq ($(and $(clean_build),$(clean_sha)),)
PACK_VERSION:=${PACK_VERSION}+git-${clean_sha}.build-${clean_build}
else ifneq ($(clean_build),)
PACK_VERSION:=${PACK_VERSION}+build-${clean_build}
else ifneq ($(clean_sha),)
PACK_VERSION:=${PACK_VERSION}+git-${clean_sha}
endif

export GOFLAGS:=$(GOFLAGS)
export CGO_ENABLED=0

all: clean verify test build

mod-tidy:
	$(GOCMD) mod tidy
	cd tools; $(GOCMD) mod tidy
	
mod-vendor:
	$(GOCMD) mod vendor
	cd tools; $(GOCMD) mod vendor
	
tidy: mod-tidy mod-vendor format

build: out
	@echo "> Building..."
	$(GOCMD) build -ldflags "-s -w -X 'github.com/buildpacks/pack.Version=${PACK_VERSION}' -extldflags ${LDFLAGS}" -trimpath -o ./out/$(PACK_BIN) -a ./cmd/pack

package: out
	tar czf ./out/$(ARCHIVE_NAME).tgz -C out/ pack

install-mockgen:
	@echo "> Installing mockgen..."
	cd tools; $(GOCMD) install github.com/golang/mock/mockgen

install-goimports:
	@echo "> Installing goimports..."
	cd tools; $(GOCMD) install golang.org/x/tools/cmd/goimports

format: install-goimports
	@echo "> Formating code..."
	@goimports -l -w -local ${PACKAGE_BASE} ${SRC}
	@go run tools/pedantic_imports/main.go ${PACKAGE_BASE} ${SRC}

install-golangci-lint:
	@echo "> Installing golangci-lint..."
	cd tools; $(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint

lint: install-golangci-lint
	@echo "> Linting code..."
	@golangci-lint run -c golangci.yaml

test: unit acceptance

# append coverage arguments
ifeq ($(TEST_COVERAGE), 1)
unit: GOTESTFLAGS:=$(GOTESTFLAGS) -coverprofile=./out/tests/coverage-unit.txt -covermode=atomic
endif
unit: out
	@echo "> Running unit/integration tests..."
	$(GOCMD) test $(GOTESTFLAGS) -timeout=$(UNIT_TIMEOUT) ./...

acceptance: out
	@echo "> Running acceptance tests..."
	$(GOCMD) test $(GOTESTFLAGS) -timeout=$(ACCEPTANCE_TIMEOUT) -tags=acceptance ./acceptance

acceptance-all: export ACCEPTANCE_SUITE_CONFIG:=$(shell cat ./acceptance/testconfig/all.json)
acceptance-all:
	@echo "> Running acceptance tests..."
	$(GOCMD) test $(GOTESTFLAGS) -timeout=$(ACCEPTANCE_TIMEOUT) -tags=acceptance ./acceptance

clean:
	@echo "> Cleaning workspace..."
	rm -rf ./out

verify: verify-format lint

generate: install-mockgen
	@echo "> Generating mocks..."
	$(GOCMD) generate ./...

verify-format: install-goimports
	@echo "> Verifying format..."
	@test -z "$(shell goimports -l -local ${PACKAGE_BASE} ${SRC})"; _err=$$?;\
	[ $$_err -ne 0 ] &&\
	echo "ERROR: Format verification failed!\n" &&\
	goimports -d -local ${PACKAGE_BASE} ${SRC} &&\
	exit $$_err;\
	exit 0;

prepare-for-pr: tidy verify test
	@git diff-index --quiet HEAD --; _err=$$?;\
	[ $$_err -ne 0 ] &&\
	echo "-----------------" &&\
	echo "NOTICE: There are some files that have not been committed." &&\
	echo "-----------------\n" &&\
	git status &&\
	echo "\n-----------------" &&\
	echo "NOTICE: There are some files that have not been committed." &&\
	echo "-----------------\n"  &&\
	exit 0;

out:
	# NOTE: Windows doesn't support `-p`
	mkdir out
	mkdir out/tests

.PHONY: clean build format imports lint test unit acceptance verify verify-format
