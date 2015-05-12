.PHONY: all build clean default

TO_BUILD := ./src/athena ./src/zeus
export GOPATH := $(shell pwd)/src/godeps:$(shell pwd)

default: build

all: build

build:
	go install -v $(TO_BUILD)

clean:
	go clean -i -v ./...
