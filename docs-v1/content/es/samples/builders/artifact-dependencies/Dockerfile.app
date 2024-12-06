ARG BASE
FROM golang:1.15-alpine as builder
...
FROM $BASE
COPY --from=builder /app .
