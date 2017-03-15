.PHONY: build test

MYPKG=$(shell go list ./... | grep -v /vendor/)

build:
	go build $(MYPKG)

test:
	go test $(MYPKG)
