FROM golang:1.17-nanoserver-1809 AS gobuild

# bake in a simple server util
COPY server.go /util/server.go
WORKDIR /util
RUN go build server.go

FROM mcr.microsoft.com/windows/nanoserver:1809

COPY --from=gobuild /util/server.exe /util/server.exe

# non-zero sets all user-owned directories to BUILTIN\Users
ENV CNB_USER_ID=1
ENV CNB_GROUP_ID=1

USER ContainerAdministrator

RUN net users /ADD pack /passwordreq:no /expires:never

LABEL io.buildpacks.stack.id=pack.test.stack
LABEL io.buildpacks.stack.mixins="[\"mixinA\", \"netcat\", \"mixin3\"]"

USER pack

# launcher requires a non-empty PATH to workaround https://github.com/buildpacks/pack/issues/800
ENV PATH c:\\Windows\\system32;C:\\Windows
