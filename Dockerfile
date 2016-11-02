FROM alpine:3.3
MAINTAINER juntaki <me@juntaki.com>

RUN mkdir /go /wiki
ENV GOPATH /go
ENV GOBIN /go/bin
WORKDIR /wiki

COPY * /wiki/
COPY style/ /wiki/style/

RUN apk add --no-cache go git && \
    go get -v && \
    go build && \
    rm -rf /go && \
    apk del go git

ENTRYPOINT ["/wiki/wiki"]
EXPOSE 8080