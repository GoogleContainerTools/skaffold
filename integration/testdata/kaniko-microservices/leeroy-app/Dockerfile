ARG BASE
FROM golang:1.16 as builder
COPY app.go .
# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /app .

FROM $BASE as target_stage
COPY --from=builder /app .

FROM busybox
