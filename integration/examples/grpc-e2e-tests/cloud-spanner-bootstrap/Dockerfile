FROM golang:1.19 as builder

WORKDIR /workspace

COPY . .

RUN go build -o  bin/cloudspannerbootstrap ./main

FROM gcr.io/distroless/base:latest
COPY --from=builder /workspace/bin/cloudspannerbootstrap cloudspannerbootstrap
ENV DATABASE=projects/test-project/instances/test-instance/databases/visitordb
ENV SPANNER_EMULATOR_HOST="cloud-spanner-emulator:9010"
ENTRYPOINT ["./cloudspannerbootstrap"]