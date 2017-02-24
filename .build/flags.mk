BENCH_FLAGS ?= -cpuprofile=cpu.pprof -memprofile=mem.pprof -benchmem
PKGS ?= $(shell glide novendor)
LIST_PKGS ?= $(shell go list ./... | grep -v /vendor/)

# Many Go tools take file globs or directories as arguments instead of packages.
PKG_FILES ?= *.go auth config dig internal metrics modules service testutils tracing ulog

# The linting tools evolve with each Go version, so run them only on the latest
# stable release.
GO_VERSION := $(shell go version | cut -d " " -f 3)

BUILD_GC_FLAGS ?= -gcflags "-trimpath=$(GOPATH)/src"

TEST_FLAGS += $(BUILD_GC_FLAGS)
ifneq ($(CI),)
RACE ?= -race
endif

