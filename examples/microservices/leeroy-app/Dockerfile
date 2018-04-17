FROM golang:1.10
WORKDIR /go/src/github.com/GoogleContainerTools/skaffold
COPY app.go .
RUN go build -o app .

FROM gcr.io/distroless/base
WORKDIR /root/
COPY --from=0 /go/src/github.com/GoogleContainerTools/skaffold .
CMD ["./app"]
