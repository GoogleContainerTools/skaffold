FROM busybox as target1

COPY file1 file2 /data/
RUN [ "$(find /data -type f | wc -l | xargs)" == "2" ]

FROM busybox as target2

COPY file3 /data/
RUN [ "$(find /data -type f | wc -l | xargs)" == "1" ]