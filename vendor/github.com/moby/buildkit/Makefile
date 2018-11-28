DESTDIR=/usr/local

binaries: FORCE
	hack/binaries

install: FORCE
	mkdir -p $(DESTDIR)/bin
	install bin/* $(DESTDIR)/bin

clean: FORCE
	rm -rf ./bin

test:
	./hack/test integration gateway dockerfile

lint:
	./hack/lint

validate-vendor:
	./hack/validate-vendor

validate-generated-files:
	./hack/validate-generated-files

validate-all: test lint validate-vendor validate-generated-files

vendor:
	./hack/update-vendor

generated-files:
	./hack/update-generated-files

.PHONY: vendor generated-files test binaries install clean lint validate-all validate-vendor validate-generated-files
FORCE:
