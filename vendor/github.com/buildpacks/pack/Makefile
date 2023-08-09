ifeq ($(OS),Windows_NT)
SHELL:=cmd.exe

# Need BLANK due to makefile parsing of `\`
# (see: https://stackoverflow.com/questions/54733231/how-to-escape-a-backslash-in-the-end-to-mean-literal-backslash-in-makefile/54733416#54733416)
BLANK:=

# Define variable named `/` to represent OS path separator (usable as `$/` in this file)
/:=\$(BLANK)
CAT=type
RMRF=rmdir /q /s
SRC=$(shell dir /q /s /b *.go | findstr /v $/out$/)
GOIMPORTS_DIFF_OPTION="-l" # Windows can't do diff-mode because it's missing the "diff" binary
PACK_BIN?=pack.exe
else
/:=/
CAT=cat
RMRF=rm -rf
SRC=$(shell find . -type f -name '*.go' -not -path "*/out/*")
GOIMPORTS_DIFF_OPTION:="-d"
PACK_BIN?=pack
endif

ACCEPTANCE_TIMEOUT?=$(TEST_TIMEOUT)
ARCHIVE_NAME=pack-$(PACK_VERSION)
GOCMD?=go
GOFLAGS?=
GOTESTFLAGS?=-v -count=1 -parallel=1
PACKAGE_BASE=github.com/buildpacks/pack
PACK_GITSHA1=$(shell git rev-parse --short=7 HEAD)
PACK_VERSION?=0.0.0
TEST_TIMEOUT?=1200s
UNIT_TIMEOUT?=$(TEST_TIMEOUT)
NO_DOCKER?=

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

BINDIR:=/usr/bin/

# this target must be listed first in order for it to be a default target,
# so that ubuntu_ppa's may be constructed using default build tools.
build: out
	@echo "> Building..."
	$(GOCMD) build -ldflags "-s -w -X 'github.com/buildpacks/pack.Version=${PACK_VERSION}' -extldflags ${LDFLAGS}" -trimpath -o ./out/$(PACK_BIN) -a ./cmd/pack

all: clean verify test build

# used by apt-get install when installing ubuntu ppa.
# move pack binary onto a path location.
install:
	mkdir -p ${DESTDIR}${BINDIR}
	cp ./out/$(PACK_BIN) ${DESTDIR}${BINDIR}/

mod-tidy:
	$(GOCMD) mod tidy
	cd tools && $(GOCMD) mod tidy

tidy: mod-tidy format

package: out
	tar czf .$/out$/$(ARCHIVE_NAME).tgz -C .$/out$/ $(PACK_BIN)

install-mockgen:
	@echo "> Installing mockgen..."
	cd tools && $(GOCMD) install github.com/golang/mock/mockgen

install-goimports:
	@echo "> Installing goimports..."
	cd tools && $(GOCMD) install golang.org/x/tools/cmd/goimports

format: install-goimports
	@echo "> Formating code..."
	@goimports -l -w -local ${PACKAGE_BASE} ${SRC}
	@go run tools/pedantic_imports/main.go ${PACKAGE_BASE} ${SRC}

install-golangci-lint:
	@echo "> Installing golangci-lint..."
	cd tools && $(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint

lint: install-golangci-lint
	@echo "> Linting code..."
	@golangci-lint run -c golangci.yaml

test: unit acceptance

# append coverage arguments
ifeq ($(TEST_COVERAGE), 1)
unit: GOTESTFLAGS:=$(GOTESTFLAGS) -coverprofile=./out/tests/coverage-unit.txt -covermode=atomic
endif
ifeq ($(NO_DOCKER),)
unit: GOTESTFLAGS:=$(GOTESTFLAGS) --tags=example
endif
unit: out
	@echo "> Running unit/integration tests..."
	$(GOCMD) test $(GOTESTFLAGS) -timeout=$(UNIT_TIMEOUT) ./...

acceptance: out
	@echo "> Running acceptance tests..."
	$(GOCMD) test $(GOTESTFLAGS) -timeout=$(ACCEPTANCE_TIMEOUT) -tags=acceptance ./acceptance

acceptance-all: export ACCEPTANCE_SUITE_CONFIG:=$(shell $(CAT) .$/acceptance$/testconfig$/all.json)
acceptance-all:
	@echo "> Running acceptance tests..."
	$(GOCMD) test $(GOTESTFLAGS) -timeout=$(ACCEPTANCE_TIMEOUT) -tags=acceptance ./acceptance

clean:
	@echo "> Cleaning workspace..."
	@$(RMRF) .$/out benchmarks.test || (exit 0)

verify: verify-format lint

generate: install-mockgen
	@echo "> Generating mocks..."
	$(GOCMD) generate ./...

verify-format: install-goimports
	@echo "> Verifying format..."
	$(if $(shell goimports -l -local ${PACKAGE_BASE} ${SRC}), @echo ERROR: Format verification failed! && goimports ${GOIMPORTS_DIFF_OPTION} -local ${PACKAGE_BASE} ${SRC} && exit 1)

prepare-for-pr: tidy verify test
	@git diff-index --quiet HEAD -- ||\
	(echo "-----------------" &&\
	echo "NOTICE: There are some files that have not been committed." &&\
	echo "-----------------\n" &&\
	git status &&\
	echo "\n-----------------" &&\
	echo "NOTICE: There are some files that have not been committed." &&\
	echo "-----------------\n"  &&\
	exit 0)

benchmark: out
	@echo "> Running Benchmarks"
	$(GOCMD) test -run=^$  -bench=. -benchtime=1s -benchmem -memprofile=./out/bench_mem.out -cpuprofile=./out/bench_cpu.out -tags=benchmarks ./benchmarks/ -v
# NOTE: You can analyze the results, using go tool pprof. For instance, you can start a server to see a graph of the cpu usage by running
# go tool pprof -http=":8082" out/bench_cpu.out. Alternatively, you can run go tool pprof, and in the ensuing cli, run
# commands like top10 or web to dig down into the cpu and memory usage
# For more, see https://blog.golang.org/pprof

# NOTE: Windows doesn't support `-p`
out:
	@mkdir out || (exit 0)
	mkdir out$/tests || (exit 0)


.PHONY: clean build format imports lint test unit acceptance prepare-for-pr verify verify-format benchmark
