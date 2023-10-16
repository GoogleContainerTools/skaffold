FROM golang:1.18 as builder
WORKDIR /code
COPY main.go .
COPY go.mod .
RUN go build -trimpath -o /app main.go

FROM alpine:3
ARG MESSAGE_ARG
ENV MESSAGE=$MESSAGE_ARG
CMD ["./app"]
COPY --from=builder /app .
