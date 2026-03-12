ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}golang:1.23 as builder

WORKDIR /workspace

COPY . .

RUN go mod tidy && go build -o bin/visitorcounter ./server

FROM gcr.io/distroless/base:latest
COPY --from=builder /workspace/bin/visitorcounter visitorcounter
ENTRYPOINT ["./visitorcounter"]