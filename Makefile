all: bunch build

build:
	bunch install
	bunch go build

bunch:
	go get -u github.com/dkulchenko/bunch
