# syntax=docker/dockerfile:1.0.0-experimental

FROM alpine:3.10

# https://github.com/tonistiigi/buildkit/blob/1604b1b9ed70bcbc002033d8dd0e65ab13f13554/client/llb/exec.go#L138
RUN --mount=type=ssh ls /run/buildkit/ssh_agent.*
