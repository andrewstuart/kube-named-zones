FROM golang

ENTRYPOINT /kube-named-zone
VOLUME /zones

RUN apt-get update && apt-get -y install bind9 && apt-get clean && rm -rf /var/lib/atp/*

ADD kube-named-zone /
