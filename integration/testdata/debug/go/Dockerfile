FROM golang:1.12 as builder

COPY app.go .
# Must use eval to handle GOGCFLAGS with spaces like `-gcflags='all=-N -l'`
ARG GOGCFLAGS
RUN eval go build "${GOGCFLAGS}" -o /app .

FROM gcr.io/distroless/base
# use GOTRACEBACK=all to trigger `skaffold debug`
ENV GOTRACEBACK=all
EXPOSE 8080
COPY --from=builder /app .
CMD ["/app"]
