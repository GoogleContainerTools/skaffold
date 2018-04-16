FROM golang:1.9.4-alpine3.7 as builder
WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold/examples/getting-started
COPY main.go .
RUN go build -o app main.go

FROM alpine:latest  
CMD ["./app"]
COPY --from=builder /go/src/github.com/GoogleCloudPlatform/skaffold/examples/getting-started/app .