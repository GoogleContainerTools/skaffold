FROM golang:1.9.2
WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold/examples/getting-started
COPY main.go .
RUN go build -o app main.go
CMD ["./app"]
