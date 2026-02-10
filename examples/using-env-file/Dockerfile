ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}golang:1.18 as builder
WORKDIR /code
COPY main.go .
COPY go.mod .
RUN go build -trimpath -o /app main.go

FROM ${BASE_PREFIX}alpine:3
ARG MESSAGE_ARG
ENV MESSAGE=$MESSAGE_ARG
CMD ["./app"]
COPY --from=builder /app .
