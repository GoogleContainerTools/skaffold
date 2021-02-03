FROM golang:1.15-alpine as builder
COPY main.go .
RUN go build -o /app main.go

FROM alpine:3 as runner
CMD ["./app"]
COPY --from=builder /app .

FROM alpine:3
# Make sure the default target is not built
RUN false
