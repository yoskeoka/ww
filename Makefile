COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
VERSION ?=
LDFLAGS := -X main.CommitHash=$(COMMIT_HASH)

ifneq ($(strip $(VERSION)),)
LDFLAGS += -X main.Version=$(VERSION)
endif

.PHONY: build test lint fmt clean perf-check perf-profile

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

perf-check:
	go run ./tools/check-performance-budget.go --budget tools/performance-budget.json

PERF_PROFILE_DIR ?= /tmp/ww-perf-profile

perf-profile:
	mkdir -p "$(PERF_PROFILE_DIR)"
	env WW_PERF_REPOS=6 WW_PERF_WORKTREES_PER_REPO=5 go test -run '^$$' -bench '^BenchmarkWorkspace(List|CleanDryRun)$$' -benchtime=1x -cpuprofile "$(PERF_PROFILE_DIR)/cpu.pprof" -memprofile "$(PERF_PROFILE_DIR)/mem.pprof" ./cmd/ww
	@echo "Profiles written to $(PERF_PROFILE_DIR)/cpu.pprof and $(PERF_PROFILE_DIR)/mem.pprof"
