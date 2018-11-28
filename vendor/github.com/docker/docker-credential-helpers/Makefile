.PHONY: all deps osxkeychain secretservice test validate wincred pass deb

TRAVIS_OS_NAME ?= linux
VERSION := $(shell grep 'const Version' credentials/version.go | awk -F'"' '{ print $$2 }')

all: test

deps:
	go get -u github.com/golang/lint/golint

clean:
	rm -rf bin
	rm -rf release

osxkeychain:
	mkdir -p bin
	go build -ldflags -s -o bin/docker-credential-osxkeychain osxkeychain/cmd/main_darwin.go

osxcodesign: osxkeychain
	$(eval SIGNINGHASH = $(shell security find-identity -v -p codesigning | grep "Developer ID Application: Docker Inc" | cut -d ' ' -f 4))
	xcrun -log codesign -s $(SIGNINGHASH) --force --verbose bin/docker-credential-osxkeychain
	xcrun codesign --verify --deep --strict --verbose=2 --display bin/docker-credential-osxkeychain

osxrelease: clean vet_osx lint fmt test osxcodesign
	mkdir -p release
	@echo "\nPackaging version ${VERSION}\n"
	cd bin && tar cvfz ../release/docker-credential-osxkeychain-v$(VERSION)-amd64.tar.gz docker-credential-osxkeychain

secretservice:
	mkdir -p bin
	go build -o bin/docker-credential-secretservice secretservice/cmd/main_linux.go

pass:
	mkdir -p bin
	go build -o bin/docker-credential-pass pass/cmd/main_linux.go

wincred:
	mkdir -p bin
	go build -o bin/docker-credential-wincred.exe wincred/cmd/main_windows.go

winrelease: clean vet_win lint fmt test wincred
	mkdir -p release
	@echo "\nPackaging version ${VERSION}\n"
	cd bin && zip ../release/docker-credential-wincred-v$(VERSION)-amd64.zip docker-credential-wincred.exe

test:
	# tests all packages except vendor
	go test -v `go list ./... | grep -v /vendor/`

vet: vet_$(TRAVIS_OS_NAME)
	go vet ./credentials

vet_win:
	go vet ./wincred

vet_osx:
	go vet ./osxkeychain

vet_linux:
	go vet ./secretservice

lint:
	for p in `go list ./... | grep -v /vendor/`; do \
		golint $$p ; \
	done

fmt:
	gofmt -s -l `ls **/*.go | grep -v vendor`

validate: vet lint fmt


BUILDIMG:=docker-credential-secretservice-$(VERSION)
deb:
	mkdir -p release
	docker build -f deb/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg DISTRO=xenial \
		--tag $(BUILDIMG) \
		.
	docker run --rm --net=none $(BUILDIMG) tar cf - /release | tar xf -
	docker rmi $(BUILDIMG)
