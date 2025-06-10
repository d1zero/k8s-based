IMAGE_NAME := d1zero/golang-app

TAG := latest

.PHONY: all
all: build push

.PHONY: build
build:
	docker build -t $(IMAGE_NAME):$(TAG) .

.PHONY: push
push:
	docker push $(IMAGE_NAME):$(TAG)
