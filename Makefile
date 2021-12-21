ifndef VERSION
VERSION := $(shell git describe --always --tags)
endif

DATE := $(shell date -u +%Y%m%d.%H%M%S)

LDFLAGS = -a -trimpath -ldflags "-X=main.version=$(VERSION)-$(DATE)"

targets = gozimhttpd gozimindex 

.PHONY: all lint test clean 

all: test $(targets)

build: $(targets)

test: testcgo testnocgo

testcgo: 
	export CGO_ENABLED=1
	export CGO_CFLAGS=$(shell pkg-config --cflags liblzma)
	go test 	

testnocgo:
	export CGO_ENABLED=0
	go test 

testnolint:
	go test -race ./...
 
lint:
	export CGO_ENABLED=0
	golangci-lint run

gozimhttpd: 
	go mod download
	export CGO_ENABLED=1
	export CGO_CFLAGS=$(shell pkg-config --cflags liblzma)
	cd cmd/gozimhttpd && go build ${LDFLAGS}	

gozimindex: 
	go mod download
	export CGO_ENABLED=1
	export CGO_CFLAGS=$(shell pkg-config --cflags liblzma)
	cd cmd/gozimindex && go build ${LDFLAGS}	

upgrade:
	go get -u -v
	go mod download
	go mod tidy
	go mod verify

clean:
	go clean
	rm -f cmd/gozimhttpd/gozimhttpd
	rm -f cmd/gozimindex/gozimindex
