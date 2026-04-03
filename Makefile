COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
VERSION ?=
LDFLAGS := -X main.CommitHash=$(COMMIT_HASH)

ifneq ($(strip $(VERSION)),)
LDFLAGS += -X main.Version=$(VERSION)
endif

.PHONY: build test lint fmt clean

build:
	go build -ldflags "$(LDFLAGS)" -o ww ./cmd/ww/

test:
	go test -short ./...

test-all:
	go test ./...

lint:
	go vet ./...
	@test -z "$$(go tool goimports -local github.com/yoskeoka/ww -l .)" || (echo "goimports check failed:"; go tool goimports -local github.com/yoskeoka/ww -l .; exit 1)

fmt:
	go tool goimports -local github.com/yoskeoka/ww -w .

clean:
	rm -f ww
