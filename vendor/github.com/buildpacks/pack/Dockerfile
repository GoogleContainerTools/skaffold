ARG base_image=gcr.io/distroless/static

FROM golang:1.22 as builder
ARG pack_version
ENV PACK_VERSION=$pack_version
WORKDIR /app
COPY . .
RUN make build

FROM ${base_image}
COPY --from=builder /app/out/pack /usr/local/bin/pack
ENTRYPOINT [ "/usr/local/bin/pack" ]
