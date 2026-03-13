COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")

.PHONY: build test lint clean

build:
	go build -ldflags "-X main.CommitHash=$(COMMIT_HASH)" -o ww ./cmd/ww/

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f ww
