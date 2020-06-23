FROM golang:1.12.9-alpine3.10 as builder
COPY hello.go .
RUN go build -o /app hello.go

FROM alpine:3.10
CMD ["./app"]
COPY --from=builder /app .
