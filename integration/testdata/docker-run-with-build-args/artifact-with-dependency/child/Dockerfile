ARG BASE_PREFIX=mirror.gcr.io/library/
ARG BASE
FROM $BASE as parent
FROM ${BASE_PREFIX}alpine
COPY --from=parent /app .
CMD ["./app"]
