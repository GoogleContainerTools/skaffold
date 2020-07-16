FROM golang:1.12 as builder

COPY app.go .
ARG SKAFFOLD_GO_GCFLAGS
ARG SKAFFOLD_LD_GCFLAGS
# Must use eval to handle gcflags with spaces like `all=-N -l`
RUN eval go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -ldflags="${SKAFFOLD_GO_LDFLAGS}" -o /app .

FROM gcr.io/distroless/base
# `skaffold debug` uses GOTRACEBACK as an indicator of the Go runtime
ENV GOTRACEBACK=all
WORKDIR /
EXPOSE 8080
COPY --from=builder /app .
CMD ["/app"]
