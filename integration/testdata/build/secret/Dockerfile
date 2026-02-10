ARG BASE_PREFIX=mirror.gcr.io/library/
# syntax=docker/dockerfile:1.0.0-experimental

FROM ${BASE_PREFIX}alpine:3

RUN --mount=type=secret,id=mysecret cat /run/secrets/mysecret

RUN --mount=type=secret,id=mysecondsecret cat /run/secrets/mysecondsecret
