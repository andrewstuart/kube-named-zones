FROM golang

MAINTAINER Andrew Stuart <andrew.stuart2@gmail.com>

ENTRYPOINT /kube-named-zones

RUN apt-get update && apt-get -y install bind9 dnsutils && apt-get clean && rm -rf /var/lib/apt/*
WORKDIR /

ADD kube-named-zones /
