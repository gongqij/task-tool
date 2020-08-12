GO=go

SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

VERSION := v0.0.1
BUILD := `git rev-parse --short HEAD`
TARGETS := task-tool-cli
project=$(PWD)
LDFLAGS += -X "$(project)/version.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "$(project)/version.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "$(project)/version.Version=$(VERSION)"
LDFLAGS += -X "$(project)/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"

build: $(TARGETS)

$(TARGETS): $(SRC)
	$(GO) build -ldflags '$(LDFLAGS)' $(project)/cmd/$@

.PHONY: clean build

clean:
	rm -f $(TARGETS)
