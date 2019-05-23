FROM golang:1.12.5-alpine3.9 as builder
COPY main.go .
RUN go build -o /app main.go

FROM alpine:3.9
CMD ["./app"]
COPY --from=builder /app .
