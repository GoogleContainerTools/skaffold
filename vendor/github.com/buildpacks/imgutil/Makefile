# Go parameters
GOCMD?=go
GOTEST=$(GOCMD) test -mod=vendor
PACKAGE_BASE=github.com/buildpacks/imgutil
PACKAGES:=$(shell $(GOCMD) list -mod=vendor ./... | grep -v /testdata/)
SRC:=$(shell find . -type f -name '*.go' -not -path "*/vendor/*")

all: test

install-goimports:
	@echo "> Installing goimports..."
	cd tools; $(GOCMD) install -mod=vendor golang.org/x/tools/cmd/goimports

format: install-goimports
	@echo "> Formating code..."
	@goimports -l -w -local ${PACKAGE_BASE} ${SRC}

vet:
	@echo "> Vetting code..."
	@$(GOCMD) vet -mod=vendor ${PACKAGES}

test: format vet
	$(GOTEST) -parallel=1 -count=1 -v ./...
