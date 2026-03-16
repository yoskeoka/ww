COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")

.PHONY: build test lint fmt clean

build:
	go build -ldflags "-X main.CommitHash=$(COMMIT_HASH)" -o ww ./cmd/ww/

test:
	go test ./...

lint:
	go vet ./...
	@test -z "$$(go tool goimports -l .)" || (echo "goimports check failed:"; go tool goimports -l .; exit 1)

fmt:
	go tool goimports -w .

clean:
	rm -f ww
