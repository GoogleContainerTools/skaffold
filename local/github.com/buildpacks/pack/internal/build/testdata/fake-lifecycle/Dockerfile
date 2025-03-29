FROM golang

RUN mkdir /lifecycle
WORKDIR /go/src/step
COPY . .
RUN GO111MODULE=on go build -o /cnb/lifecycle/phase ./phase.go

ENV CNB_USER_ID 111
ENV CNB_GROUP_ID 222

LABEL io.buildpacks.stack.id="test.stack"
LABEL io.buildpacks.builder.metadata="{\"buildpacks\":[{\"id\":\"just/buildpack.id\",\"version\":\"1.2.3\"}],\"lifecycle\":{\"version\":\"0.5.0\",\"api\":{\"buildpack\":\"0.2\",\"platform\":\"0.1\"}}}"