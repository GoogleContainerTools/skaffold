FROM golang:1.10.1-alpine3.7

WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold
CMD ["./app"]
COPY main.go .
RUN go build -o app main.go
