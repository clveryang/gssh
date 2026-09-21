BIN := gssh
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test fmt vet install clean

build:
	go build -ldflags "-X github.com/clveryang/gssh/cmd.Version=$(VERSION)" -o $(BIN) .

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

install: build
	install -m 0755 $(BIN) $(HOME)/.local/bin/$(BIN)

clean:
	rm -f $(BIN)
