ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}golang:1.15-alpine as builder
COPY web.go .
RUN go build -o /web .

FROM ${BASE_PREFIX}alpine:3
CMD ["./web"]
COPY --from=builder /web .
