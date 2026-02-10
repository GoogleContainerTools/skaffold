ARG BASE_PREFIX=mirror.gcr.io/library/
FROM ${BASE_PREFIX}busybox

COPY file1 file2 /data/
RUN [ "$(find /data -type f | wc -l | xargs)" == "2" ]