FROM busybox
ARG IMAGE_TAG
RUN echo $IMAGE_TAG
COPY "script-${IMAGE_TAG}.sh" script.sh
CMD ["/bin/sh","script.sh"]
