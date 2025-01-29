FROM golang:1.23 as builder
WORKDIR /code
COPY main.go .
COPY go.mod .
# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
ARG IMAGE_REPO
ARG IMAGE_NAME
ARG IMAGE_TAG
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -ldflags="-X main.ImageRepo=${IMAGE_REPO} -X main.ImageName=${IMAGE_NAME} -X main.ImageTag=${IMAGE_TAG}" -trimpath -o /app main.go

