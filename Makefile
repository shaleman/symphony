.PHONY: all build clean default checks

TO_BUILD := ./athena/ ./zeus/
TO_TEST := ./zeus/... ./pkg/...
TO_LINT := ./zeus/ ./athena/ ./pkg/altaspec ./pkg/cephdriver ./pkg/libdocker ./pkg/libfsm ./pkg/netutils ./pkg/ovsdriver ./pkg/ovsdriver/ovsdbDump ./pkg/psutil ./pkg/rsrcMgr

default: build

all: build

godep:
	go get github.com/kr/godep

checks:
	./checks "$(TO_LINT)"

build: godep checks
	godep go install -v $(TO_BUILD)

clean: godep
	godep go clean -i -v ./...

test: godep
	@if [ `id -u` -ne 0 ]; then echo -e "\n\n\tYou must be root\n\n"; exit 1; fi
	godep go test -v $(TO_TEST)

save: godep
	godep save ./zeus ./athena
