# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

COPY main.go .
RUN go build -o /app main.go

FROM alpine:3

COPY --from=builder /app .

CMD ["/app"]
