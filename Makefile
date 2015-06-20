.PHONY: all build clean default

TO_BUILD := ./athena/ ./zeus/
TO_TEST := ./zeus/... ./pkg/...

default: build

all: build

godep:
	go get github.com/kr/godep

build: godep
	godep go install -v $(TO_BUILD)

clean: godep
	godep go clean -i -v ./...

test: godep
	godep go test -v $(TO_TEST)
