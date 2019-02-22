FROM golang:1.9.4-alpine3.7

WORKDIR /go/src/github.com/GoogleContainerTools/skaffold/examples/getting-started
CMD ["./app"]
COPY main.go .
RUN go build -o app main.go
