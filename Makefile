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
	@if [ `id -u` -ne 0 ]; then echo -e "\n\n\tYou must be root\n\n"; exit 1; fi
	godep go test -v $(TO_TEST)
