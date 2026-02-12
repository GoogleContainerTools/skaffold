ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}golang:1.15-alpine as builder
COPY main.go .
RUN go build -o /app main.go

FROM ${BASE_PREFIX}alpine:3 as runner
CMD ["./app"]
COPY --from=builder /app .

FROM ${BASE_PREFIX}alpine:3
# Make sure the default target is not built
RUN false
