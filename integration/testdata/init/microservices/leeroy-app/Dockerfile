FROM golang:1.15-alpine as builder
COPY app.go .
RUN go build -o /app .

FROM mirror.gcr.io/library/alpine:3
CMD ["./app"]
COPY --from=builder /app .
