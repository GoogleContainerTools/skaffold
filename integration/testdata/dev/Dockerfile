FROM busybox

# SLEEP is to simulate slow builds, no sleep by default
ARG SLEEP=0

COPY foo /foo

ENV SLEEP_TIMEOUT=${SLEEP}
RUN echo "sleep ${SLEEP_TIMEOUT}"
RUN [ "$(cat /foo)" == "foo" ] && sleep ${SLEEP_TIMEOUT} || echo "no sleep!"

CMD while true; do cat /foo; sleep 1; done
