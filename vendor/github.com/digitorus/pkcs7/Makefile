all: vet test

test:
	go test -covermode=count -coverprofile=coverage.out .

test-legacy:
	GODEBUG=x509sha1=1 go test -tags=legacy -covermode=count -coverprofile=coverage.out .

showcoverage: test
	go tool cover -html=coverage.out

vet:
	go vet .

golangci-lint:
	golangci-lint run

gettools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
