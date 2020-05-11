FROM golang:1.12.9-alpine3.10 as builder
COPY app.go .
RUN go build -o /app .

FROM alpine:3.10 as target_stage
CMD ["./app"]
COPY --from=builder /app .

FROM busybox
