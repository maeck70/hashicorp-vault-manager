.PHONY: all build test clean run

BINARY=bin/vault-tui

all: test build

build:
	@mkdir -p bin
	go build -o $(BINARY) .

test:
	go test -v ./...

run: build
	./$(BINARY)

clean:
	rm -rf bin/
