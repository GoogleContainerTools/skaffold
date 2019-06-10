FROM golang:1.12 as builder

COPY app.go .
# disable optimizations for debugging
RUN go build -gcflags='all=-N -l' -o /app .

FROM gcr.io/distroless/base
# add GOTRACEBACK=all to trigger `skaffold debug`
ENV GOTRACEBACK=all
EXPOSE 8080
COPY --from=builder /app .
CMD ["/app"]
