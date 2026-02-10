ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}golang:1.15-alpine as builder
COPY main.go .
RUN go build -o /main .

FROM ${BASE_PREFIX}alpine:3
CMD ["./main"]
COPY --from=builder /main .
