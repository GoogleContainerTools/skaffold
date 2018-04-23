FROM golang:1.10.1-alpine3.7 as builder
COPY web.go .
RUN go build -o /web .

FROM gcr.io/distroless/base
CMD ["./web"]
COPY --from=builder /web .
