FROM golang:1.12 as builder

COPY app.go .
ARG SKAFFOLD_GO_GCFLAGS
RUN go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /app .

FROM gcr.io/distroless/base
# `skaffold debug` uses GOTRACEBACK as an indicator of the Go runtime
ENV GOTRACEBACK=all
WORKDIR /
EXPOSE 8080
COPY --from=builder /app .
CMD ["/app"]
