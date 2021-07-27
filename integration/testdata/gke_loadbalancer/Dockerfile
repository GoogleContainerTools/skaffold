FROM golang:1.16-alpine as builder
COPY main.go .
RUN go build -o /main .

FROM alpine:3
CMD ["./main"]
COPY --from=builder /main .
