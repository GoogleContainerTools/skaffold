FROM mcr.microsoft.com/windows/nanoserver:1809

# non-zero sets all user-owned directories to BUILTIN\Users
ENV CNB_USER_ID=1
ENV CNB_GROUP_ID=1

USER ContainerAdministrator

RUN net users /ADD pack /passwordreq:no /expires:never

LABEL io.buildpacks.stack.id=pack.test.stack
LABEL io.buildpacks.stack.mixins="[\"mixinA\", \"build:mixinTwo\", \"netcat\", \"mixin3\"]"

USER pack
