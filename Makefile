.PHONY: build clean test

BINARY := bin/rgbtocmyk

build:
	CGO_ENABLED=1 go build -o $(BINARY) ./cmd/rgbtocmyk

clean:
	rm -rf bin/

test:
	CGO_ENABLED=1 go test ./...
