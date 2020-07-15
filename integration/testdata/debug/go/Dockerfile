FROM golang:1.12 as builder

COPY app.go .
# Must use eval to handle GOGCFLAGS with spaces like `-gcflags='all=-N -l'`
ARG GO_GCFLAGS
RUN eval go build --gcflags="${GO_GCFLAGS}" -o /app .

FROM gcr.io/distroless/base
# `skaffold debug` uses GOTRACEBACK as an indicator of the Go runtime
ENV GOTRACEBACK=all
WORKDIR /
EXPOSE 8080
COPY --from=builder /app .
CMD ["/app"]
