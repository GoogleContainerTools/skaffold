FROM golang:1.10.1-alpine3.7 as builder
COPY app.go .
RUN go build -o /app .

FROM gcr.io/distroless/base
CMD ["./app"]
COPY --from=builder /app .
