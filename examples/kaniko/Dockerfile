FROM golang:1.9.4-alpine3.7

WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold/examples/kaniko
CMD ["./app"]
COPY main.go .
RUN go build -o app main.go
