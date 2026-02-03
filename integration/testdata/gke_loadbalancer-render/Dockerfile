FROM golang:1.15-alpine as builder
COPY main.go .
RUN go build -o /main .

FROM mirror.gcr.io/library/alpine:3
CMD ["./main"]
COPY --from=builder /main .
