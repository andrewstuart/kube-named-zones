FROM golang

MAINTAINER Andrew Stuart <andrew.stuart2@gmail.com>

CMD /go/src/kube-named-zones

RUN apt-get update && apt-get -y install bind9 dnsutils && apt-get clean && rm -rf /var/lib/apt/*

ADD . /go
RUN go get && chown -R 1000 .

USER 1000
