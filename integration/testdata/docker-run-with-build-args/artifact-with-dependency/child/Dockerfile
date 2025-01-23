ARG BASE
FROM $BASE as parent
FROM alpine
COPY --from=parent /app .
CMD ["./app"]
