FROM golang:1.15-alpine as builder
COPY main.go .
RUN go build -o /app main.go

FROM alpine:3
CMD ["./app"]
COPY --from=builder /app .
