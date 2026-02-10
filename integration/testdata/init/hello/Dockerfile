ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}golang:1.15-alpine as builder
COPY hello.go .
RUN go build -o /app hello.go

FROM ${BASE_PREFIX}alpine:3
CMD ["./app"]
COPY --from=builder /app .
