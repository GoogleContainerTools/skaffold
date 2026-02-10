FROM golang:1.25 as builder

WORKDIR /workspace

COPY . .

RUN go build -o  bin/visitorcounter ./server

FROM gcr.io/distroless/base:latest
COPY --from=builder /workspace/bin/visitorcounter visitorcounter
ENTRYPOINT ["./visitorcounter"]
