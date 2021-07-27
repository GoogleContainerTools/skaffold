FROM golang:1.16-alpine as builder
COPY app.go .
RUN go build -o /app .

FROM alpine:3
CMD ["./app"]
COPY --from=builder /app .
