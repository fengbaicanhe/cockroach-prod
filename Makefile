all: bunch build check

build:
	bunch install
	bunch go build

bunch:
	go get github.com/dkulchenko/bunch

GOFILES := $(shell find . -name '*.go' | grep -vF '/.')

check:
	bunch exec errcheck -ignore 'github.com/spf13/cobra:Usage' ./...
	bunch exec go-nyet $(GOFILES)
	bunch exec golint ./...
	bunch exec gofmt -s -l $(GOFILES)
	bunch exec goimports -l $(GOFILES)
	go vet ./...
