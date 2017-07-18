NAME    := kafka_zookeeper_exporter
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

.build/dep.ok:
	go install github.com/golang/dep/cmd/dep
	@mkdir -p .build
	touch $@

.PHONY: vendor
vendor: .build/dep.ok
	dep ensure
	dep prune

.PHONY: vendor-update
vendor-update: .build/dep.ok
	dep ensure -update
	dep prune

.PHONY: all
all: lint test build
