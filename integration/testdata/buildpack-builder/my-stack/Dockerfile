FROM golang:1.12.9-alpine3.10

RUN apk add --no-cache ca-certificates bash

ARG cnb_uid=1000
ARG cnb_gid=1000
ARG stack_id="my-stack"

LABEL io.buildpacks.stack.id=${stack_id}

ENV CNB_USER_ID=${cnb_uid}
ENV CNB_GROUP_ID=${cnb_gid}
ENV CNB_STACK_ID=${stack_id}

RUN addgroup -S -g ${cnb_gid} cnb && adduser -S -G cnb -u ${cnb_gid} cnb
USER cnb
