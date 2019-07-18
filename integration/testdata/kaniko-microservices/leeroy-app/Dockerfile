FROM golang:1.12.6-alpine3.9 as builder
COPY app.go .
RUN go build -o /app .

FROM alpine:3.9 as target_stage
CMD ["./app"]
COPY --from=builder /app .

FROM busybox
