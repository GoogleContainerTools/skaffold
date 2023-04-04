FROM buildpack-deps:focal-scm
ARG GO_VERSION

# Provided automatically by docker buildx.
ARG TARGETPLATFORM
ARG BUILDPLATFORM

ENV PATH /usr/local/go/bin:/usr/local/protoc/bin:$PATH

RUN arch=$(echo $TARGETPLATFORM | sed 's|\/|-|') && wget -O go.tgz "https://dl.google.com/go/go${GO_VERSION}.${arch}.tar.gz" && tar -C /usr/local -xzf go.tgz && rm go.tgz;
COPY requirements.txt /tmp/requirements.txt
COPY boulder.rsyslog.conf /etc/rsyslog.d/
COPY build.sh /tmp/build.sh
RUN /tmp/build.sh
RUN sed -i '/imklog/s/^/#/' /etc/rsyslog.conf
RUN sed -i '/$ActionFileDefaultTemplate/s/^/#/' /etc/rsyslog.conf
RUN sed -i '/$RepeatedMsgReduction on/s/^/#/' /etc/rsyslog.conf
