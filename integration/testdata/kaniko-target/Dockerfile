FROM golang:1.12.5-alpine3.9 as builder
COPY main.go .
RUN go build -o /app main.go

FROM alpine:3.9 as runner
CMD ["./app"]
COPY --from=builder /app .

FROM alpine:3.9
# Make sure the default target is not built
RUN false
