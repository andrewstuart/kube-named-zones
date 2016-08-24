REG=docker.astuart.co:5000
IMAGE=kube-named-zones

.PHONY: build push deploy

TAG=$(REG)/$(IMAGE)

build:
	go build
	upx $(IMAGE)
	docker build -t $(TAG) .

push: build
	docker push $(TAG)

deploy: push
	kubectl delete po -l app=$(IMAGE)
