FROM golang:1.12.6-alpine3.9 as builder
COPY main.go .
RUN go build -o /main .

FROM alpine:3.9
CMD ["./main"]
COPY --from=builder /main .
