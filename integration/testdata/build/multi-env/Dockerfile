FROM busybox

ENV file1=file1 \
    file2=file2

COPY $file1 $file2 /data/
RUN [ "$(find /data -type f | wc -l | xargs)" == "2" ]