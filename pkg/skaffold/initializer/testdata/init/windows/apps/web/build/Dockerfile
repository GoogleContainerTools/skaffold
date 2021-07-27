FROM golang:1.16-alpine as builder
COPY web.go .
RUN go build -o /web .

FROM alpine:3
CMD ["./web"]
COPY --from=builder /web .
