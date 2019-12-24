GOCMD?=go
GOENV=CGO_ENABLED=0
GOFLAGS?=-mod=vendor
PACK_VERSION?=dev-$(shell date +%Y-%m-%d-%H:%M:%S)
PACK_BIN?=pack
PACKAGE_BASE=github.com/buildpacks/pack
PACKAGES:=$(shell $(GOCMD) list ./... | grep -v /testdata/)
SRC:=$(shell find . -type f -name '*.go' -not -path "*/vendor/*")
ARCHIVE_NAME=pack-$(PACK_VERSION)

export GOFLAGS:=$(GOFLAGS)

all: clean verify test build

build:
	@echo "> Building..."
	mkdir -p ./out
	$(GOENV) $(GOCMD) build -ldflags "-X 'github.com/buildpacks/pack/cmd.Version=${PACK_VERSION}'" -o ./out/$(PACK_BIN) -a ./cmd/pack

package:
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

install-golangci-lint:
	@echo "> Installing golangci-lint..."
	cd tools; $(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint

lint: install-golangci-lint
	@echo "> Linting code..."
	@golangci-lint run -c golangci.yaml

test: unit acceptance

unit: format lint
	@echo "> Running unit/integration tests..."
	$(GOCMD) test -v -count=1 -parallel=1 -timeout=0 ./...

acceptance: format lint
	@echo "> Running acceptance tests..."
	$(GOCMD) test -v -count=1 -parallel=1 -timeout=0 -tags=acceptance ./acceptance

acceptance-all: format lint
	@echo "> Running acceptance tests..."
	ACCEPTANCE_SUITE_CONFIG=$$(cat ./acceptance/testconfig/all.json) $(GOCMD) test -v -count=1 -parallel=1 -timeout=0 -tags=acceptance ./acceptance

clean:
	@echo "> Cleaning workspace..."
	rm -rf ./out

verify: verify-format lint

generate: install-mockgen
	@echo "> Generating mocks..."
	$(GOCMD) generate ./...

verify-format: install-goimports
	@echo "> Verifying format..."
	@test -z "$(shell goimports -l -local ${PACKAGE_BASE} ${SRC})";\
	_err=$$?;\
	[ $$_err -ne 0 ] && echo "ERROR: Format verification failed!\n" &&\
	goimports -d -local ${PACKAGE_BASE} ${SRC} && exit $$_err;\
	exit 0;

.PHONY: clean build format imports lint test unit acceptance verify verify-format