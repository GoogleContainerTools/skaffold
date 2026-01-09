# Stage 1
FROM debian:stable-slim AS builder
COPY --from=golang:1.18 /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /code
COPY main.go .
COPY go.mod .

# `skaffold debug` sets SKAFFOLD_GO_GCFLAGS to disable compiler optimizations
ARG SKAFFOLD_GO_GCFLAGS
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -trimpath -o /app main.go

# Stage 2
FROM debian:stable-slim
ENV GOTRACEBACK=single

WORKDIR /
COPY --from=builder /app .
CMD ["./app"]