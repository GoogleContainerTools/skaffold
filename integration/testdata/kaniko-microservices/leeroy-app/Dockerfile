ARG BASE_PREFIX=mirror.gcr.io/library/
ARG BASE
FROM ${BASE_PREFIX}golang:1.15 as builder
COPY app.go .
# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /app .

FROM $BASE as target_stage
COPY --from=builder /app .

FROM ${BASE_PREFIX}busybox
