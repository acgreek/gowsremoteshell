IMGHREF=testshell
TAG?=$(shell git describe --tags --always)
GO?=go
IMAGE=$(IMGHREF)

all: remoteshell_client remoteshell_server

remoteshell_server: server/server.go
	go build -o $@ $<

remoteshell_server-linux: server/server.go
	env GOOS=linux go build -o $@ $<

remoteshell_client: client/cancel_reader.go client/websocketcmd.go
	go build -o $@ client/cancel_reader.go client/websocketcmd.go

image:
	docker build -f Dockerfile $(DOCKER_ARGS) -t=$(IMAGE):$(TAG) -t "$(IMAGE):latest" .

deploy: image
	docker run -d -p 9070:9070  docker.io/library/testshell:1 /remoteshell_server-linux

.PHONY: image deploy all
