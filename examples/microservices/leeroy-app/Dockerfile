FROM golang:1.10
WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold
COPY app.go .
RUN go build -o app .

FROM scratch
WORKDIR /root/
COPY --from=0 /go/src/github.com/GoogleCloudPlatform/skaffold .
CMD ["./app"]
