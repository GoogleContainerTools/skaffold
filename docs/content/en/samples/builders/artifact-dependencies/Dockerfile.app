ARG BASE
FROM golang:1.12.9-alpine3.10 as builder
...
FROM $BASE
COPY --from=builder /app .