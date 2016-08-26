REG=andrewstuart
BINARY?=kube-named-zones
IMAGE:=$(BINARY)

.PHONY: build push deploy

TAG=$(REG)/$(IMAGE)

build:
	docker build -t $(TAG) .

push: build
	docker push $(TAG)

deploy: push
	kubectl delete po -l app=$(IMAGE)
