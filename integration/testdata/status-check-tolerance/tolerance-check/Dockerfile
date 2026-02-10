ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}alpine:latest
ARG STOP_FAILING_TIME
ENV STOP_FAILING_TIME=${STOP_FAILING_TIME:-not_found}  


COPY script.sh /script.sh
ENTRYPOINT [ "/script.sh" ]