NAME    := kafka-zookeeper-exporter
VERSION := $(shell git describe --tags --always --dirty='-dev')

.PHONY: build
build:
	go build -ldflags "-X main.version=$(shell git describe --tags --always --dirty=-dev)"

.PHONY: build-docker
build-docker:
	docker build --build-arg VERSION=$(VERSION) -t $(NAME):$(VERSION) .

.PHONY: test
test: lint
	go test $(go list ./... | grep -v /vendor/)

.PHONY: lint
lint:
	golint ./... | (egrep -v ^vendor/ || true)

.PHONY: vendor
vendor:
	govendor fetch +m +e +v

.PHONY: all
all: lint test build
