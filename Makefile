.PHONY: test

MYPKG=$(shell go list ./... | grep -v /vendor/)

test:
	go test $(MYPKG)
