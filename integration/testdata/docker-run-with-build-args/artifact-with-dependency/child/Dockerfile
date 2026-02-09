ARG BASE
FROM $BASE as parent
FROM mirror.gcr.io/library/alpine
COPY --from=parent /app .
CMD ["./app"]
