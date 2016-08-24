FROM golang

ENTRYPOINT /kube-named-zone

RUN apt-get update && apt-get -y install bind9 dnsutils && apt-get clean && rm -rf /var/lib/apt/*
WORKDIR /

ADD kube-named-zones /
