FROM alpine

RUN apk add hugo

ADD . /workspace

WORKDIR /workspace

EXPOSE 8080


CMD ["sh", "-c", "hugo serve -D --bind=0.0.0.0 --port=8080 --baseURL=${BASE_URL} --appendPort=false"]
