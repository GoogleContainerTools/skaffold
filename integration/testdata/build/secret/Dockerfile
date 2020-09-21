# syntax=docker/dockerfile:1.0.0-experimental

FROM alpine:3.10

RUN --mount=type=secret,id=mysecret cat /run/secrets/mysecret
