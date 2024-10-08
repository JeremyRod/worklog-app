VERSION := 1.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD)

build:
    go build -ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT)" -o worklog

.PHONY: build