ifndef VERSION
VERSION := $(shell git describe --always --tags)
endif

DATE := $(shell date -u +%Y%m%d.%H%M%S)

LDFLAGS = -a -trimpath -ldflags "-X=main.version=$(VERSION)-$(DATE)"

targets = gozimhttpd gozimindex 

.PHONY: all lint test clean 

all: test $(targets)

test: testcgo testnocgo

testcgo: export CGO_ENABLED = 1
testcgo: export CGO_CFLAGS = $(shell pkg-config --cflags liblzma)
testcgo:
	go test 

testnocgo: export CGO_ENABLED = 0
testnocgo:
	go test 

testnolint:
	go test -race ./...


lint: export CGO_ENABLED = 0
lint:
	golangci-lint run

gozimhttpd:
	cd cmd/gozimhttpd && go build $LDFLAGS

gozimindex:
	cd cmd/gozimindex && go build $LDFLAGS

clean:
	rm -f cmd/gozimhttpd/gozimhttpd
	rm -f cmd/gozimindex/gozimindex
