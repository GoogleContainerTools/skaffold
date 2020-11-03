ARG BASE
FROM golang:1.12.9-alpine3.10 as builder
COPY main.go .

RUN go build -o /app .

FROM $BASE
COPY --from=builder /app .
