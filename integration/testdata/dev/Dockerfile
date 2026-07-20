ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}busybox
COPY foo /foo
CMD while true; do cat /foo; sleep 1; done
