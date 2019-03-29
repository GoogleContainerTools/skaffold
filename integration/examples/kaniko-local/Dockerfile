FROM golang:1.10.1-alpine3.7 as builder
COPY main.go .
RUN go build -o /app main.go

FROM alpine:3.7  
CMD ["./app"]
COPY --from=builder /app .
