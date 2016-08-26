# FROM golang
# MAINTAINER Andrew Stuart <andrew.stuart2@gmail.com>

FROM golang:onbuild
RUN apt-get update && apt-get -y install bind9 dnsutils && apt-get clean && rm -rf /var/lib/apt/*

# RUN mkdir -p /go/src/kube-named-zones
# WORKDIR /go/src/kube-named-zones

# CMD ["go-wrapper", "run"]

# COPY . /go/src/kube-named-zones
# RUN go-wrapper download && go-wrapper install
