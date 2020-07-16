FROM golang:1.12 as builder
ARG SKAFFOLD_GO_GCFLAGS

COPY app.go .
RUN echo "SKAFFOLD_GO_GCFLAGS: ${SKAFFOLD_GO_GCFLAGS}"
# Must use eval to handle GOGCFLAGS with spaces like `all=-N -l`
RUN eval go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /app .

FROM gcr.io/distroless/base
# `skaffold debug` uses GOTRACEBACK as an indicator of the Go runtime
ENV GOTRACEBACK=all
WORKDIR /
EXPOSE 8080
COPY --from=builder /app .
CMD ["/app"]
