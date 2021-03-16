FROM busybox
COPY foo /

# ensure that the file content of foo is not "foo" when doing a build
RUN echo "bar" > /foo

CMD while true; do cat /foo && sleep 5; done
