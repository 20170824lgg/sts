all: build

build:
	go build

docker: build
	docker build -t 20170824lgg/sts:latest .

.PHONY: all build docker
