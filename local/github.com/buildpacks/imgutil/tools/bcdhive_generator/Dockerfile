ARG go_version
FROM golang:${go_version}-bullseye

RUN apt update \
    && apt install -y libhivex-dev libhivex-bin libwin-hivex-perl

COPY . /src

WORKDIR /src

RUN go generate ./ \
    && go test -parallel=1 -count=1 -v . \
    && go install .

ENTRYPOINT ["/go/bin/bcdhive_gen"]
