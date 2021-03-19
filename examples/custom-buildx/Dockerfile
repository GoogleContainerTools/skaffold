# As Go supports cross-compilation, `--platform=$BUILDPLATFORM`
# results in a dramatic speed-up as the build runs natively
# instead of using emulation.
# BUILD{PLATFORM,OS,ARCH} are set to your docker engine's platform
# TARGET{PLATFORM,OS,ARCH} are set to the desired platform
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

COPY main.go .

# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /app main.go

FROM alpine:3
# Define GOTRACEBACK to mark this container as using the Go language runtime
# for `skaffold debug` (https://skaffold.dev/docs/workflows/debug/).
ENV GOTRACEBACK=single
CMD ["./app"]
COPY --from=builder /app .
