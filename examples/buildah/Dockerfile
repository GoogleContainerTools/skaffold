# Build the server binary
FROM docker.io/library/golang:1.17 as builder

WORKDIR /workspace
# Copy the go source
COPY main.go main.go


# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -a -o binary main.go

FROM docker.io/library/alpine
ENV GOTRACEBACK=all
WORKDIR /app

COPY --from=builder /workspace/binary .

USER 999:999

ENTRYPOINT ["/app/binary"]