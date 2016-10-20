FROM alpine:3.3
MAINTAINER juntaki <me@juntaki.com>

RUN apk add --no-cache go git

RUN mkdir /go
ENV GOPATH /go
ENV GOBIN /go/bin

RUN mkdir /wiki
WORKDIR /wiki

COPY * /wiki/
COPY style/ /wiki/style/

RUN go get -v
RUN go build

ENTRYPOINT ["/wiki/wiki"]
EXPOSE 8080