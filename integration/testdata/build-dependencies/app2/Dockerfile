FROM busybox

# SLEEP is to simulate build time
ARG SLEEP=0
# FAIL=1 will cause the build to fail
ARG FAIL=0
COPY foo /foo

ENV SLEEP_TIMEOUT=${SLEEP}
ENV FAIL=${FAIL}
RUN echo "sleep ${SLEEP_TIMEOUT}"
RUN sleep ${SLEEP_TIMEOUT}
RUN [ "${FAIL}" == "0" ] || false

CMD while true; do cat /foo; sleep 1; done