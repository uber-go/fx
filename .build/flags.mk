BENCH_FLAGS ?= -cpuprofile=cpu.pprof -memprofile=mem.pprof -benchmem
PKGS ?= $(shell glide novendor)
LIST_PKGS ?= $(shell go list ./... | grep -v /vendor/)

# Many Go tools take file globs or directories as arguments instead of packages.
ROOT_PKG_FILES := $(wildcard *.go)
# Get all the directories in fx root that are not vendor, hidden, or examples, join into one line
PKG_NAMES := $(shell find . -d 1 -type d | cut -f 2 -d"/" | grep -ve "vendor\|\.\|examples" | paste -sd " " -)
PKG_FILES ?= $(ROOT_PACKAGE_FILES) $(PKG_NAMES)

# The linting tools evolve with each Go version, so run them only on the latest
# stable release.
GO_VERSION := $(shell go version | cut -d " " -f 3)

BUILD_GC_FLAGS ?= -gcflags "-trimpath=$(GOPATH)/src"

TEST_FLAGS += $(BUILD_GC_FLAGS)
ifneq ($(CI),)
RACE ?= -race
endif

