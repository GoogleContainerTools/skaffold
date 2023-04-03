TAGS ?= ""
GO_BIN ?= "go"


install: tidy
	cd ./cmd/here && $(GO_BIN) install -tags ${TAGS} -v .
	make tidy

tidy:
	$(GO_BIN) mod tidy -v

build: tidy
	$(GO_BIN) build -v .
	make tidy

test: tidy
	$(GO_BIN) test -count 1 -cover -tags ${TAGS} -timeout 10s ./...
	make tidy

cov:
	$(GO_BIN) test -coverprofile cover.out -count 1 -tags ${TAGS} ./...
	go tool cover -html cover.out
	make tidy

ci-test:
	$(GO_BIN) test -tags ${TAGS} -race ./...

lint:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint
	golangci-lint run --enable-all
	make tidy

update:
	rm go.*
	$(GO_BIN) mod init
	$(GO_BIN) mod tidy
	make test
	make install
	make tidy

release-test:
	$(GO_BIN) test -tags ${TAGS} -race ./...
	make tidy

release:
	make tidy
	release -y -f version.go --skip-packr
	make tidy


