FROM golang:1.10 as build
WORKDIR /usr/local/go/src/github.com/dgageot/hexagons/
COPY vendor vendor
RUN find vendor -maxdepth 2 -mindepth 2 -type d -exec sh -c 'go install -i github.com/dgageot/hexagons/{}/... || true' \;
COPY main.go .
RUN go build -o /api main.go

FROM gcr.io/distroless/base
EXPOSE 8080
ENTRYPOINT ["/api"]
COPY --from=build /api /
