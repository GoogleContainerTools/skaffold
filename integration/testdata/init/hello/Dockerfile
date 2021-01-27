FROM golang:1.15-alpine as builder
COPY hello.go .
RUN go build -o /app hello.go

FROM alpine:3
CMD ["./app"]
COPY --from=builder /app .
