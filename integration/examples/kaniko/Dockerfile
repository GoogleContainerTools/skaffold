FROM gcr.io/google-appengine/golang

WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold
CMD ["./app"]
COPY main.go .
RUN go build -o app main.go
