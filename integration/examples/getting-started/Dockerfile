FROM golang:1.12.9-alpine3.10 as builder

# These flag values are provided when building for debug
ARG SKAFFOLD_GO_GCFLAGS
ARG SKAFFOLD_GO_LDFLAGS

COPY main.go .
RUN eval go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -ldflags="${SKAFFOLD_GO_LDFLAGS}" -o /app main.go

FROM alpine:3.10
# Define GOTRACEBACK to mark this container as using the Go language runtime
# for `skaffold debug` (https://skaffold.dev/docs/workflows/debug/).
ENV GOTRACEBACK=single
CMD ["./app"]
COPY --from=builder /app .
