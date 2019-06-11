FROM golang:1.12 as builder

COPY app.go .
# Seemingly must assign the ARG to an environment variable for proper handling
# of args with spaces like `-gcflags='all=-N -l'`
ARG GOBUILDFLAGS
ENV GOBUILDFLAGS=${GOBUILDARGS}
RUN go build ${GOBUILDFLAGS} -o /app .

FROM gcr.io/distroless/base
# use GOTRACEBACK=all to trigger `skaffold debug`
ENV GOTRACEBACK=all
EXPOSE 8080
COPY --from=builder /app .
CMD ["/app"]
