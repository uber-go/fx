SHELL := /bin/bash
PROJECT_ROOT := go.uber.org/fx

SUPPORT_FILES := .build
include $(SUPPORT_FILES)/colors.mk
include $(SUPPORT_FILES)/deps.mk
include $(SUPPORT_FILES)/flags.mk
include $(SUPPORT_FILES)/verbosity.mk

.PHONY: all
all: lint test
.DEFAULT_GOAL := all

COV_REPORT := overalls.coverprofile

# all .go files that don't exist in hidden directories
ALL_SRC := $(shell find . -name "*.go" | grep -v -e vendor \
	-e ".*/\..*" \
	-e "examples/keyvalue/.*" \
	-e ".*/_.*")

TEST_TIMEOUT := "-timeout=3s"

.PHONY: test
test: examples
	$(ECHO_V)go test $(RACE) $(TEST_TIMEOUT) $(PKGS)
	$(ECHO_V)$(MAKE) $(COV_REPORT)

TEST_IGNORES = vendor .git
COVER_IGNORES = $(TEST_IGNORES) examples testutils

comma := ,
null :=
space := $(null) #
OVERALLS_IGNORE = $(subst $(space),$(comma),$(strip $(COVER_IGNORES)))

ifeq ($(V),0)
_FILTER_OVERALLS = cat
else
_FILTER_OVERALLS = grep -v "^Processing:"
endif

# This is the default for overalls
COVER_OUT := profile.coverprofile

$(COV_REPORT): $(PKG_FILES) $(ALL_SRC)
	@$(call label,Running tests)
	@echo
	$(ECHO_V)$(OVERALLS) -project=$(PROJECT_ROOT) \
		-ignore "$(OVERALLS_IGNORE)" \
		-covermode=atomic \
		$(DEBUG_FLAG) -- \
		$(TEST_FLAGS) $(RACE) $(TEST_TIMEOUT) $(TEST_VERBOSITY_FLAG) | \
		grep -v "No Go Test files" | \
		$(_FILTER_OVERALLS)
	$(ECHO_V)if [ -a $(COV_REPORT) ]; then \
		$(GOCOV) convert $@ | $(GOCOV) report ; \
	fi

COV_HTML := coverage.html

$(COV_HTML): $(COV_REPORT)
	$(ECHO_V)$(GOCOV) convert $< | gocov-html > $@

.PHONY: coveralls
coveralls: $(COV_REPORT)
	$(ECHO_V)goveralls -service=travis-ci -coverprofile=overalls.coverprofile

BENCH ?= .
BENCH_FILE ?= .bench/new.txt
.PHONY: bench
bench:
	@$(call label,Running benchmarks)
	$(ECHO_V)rm -f $(BENCH_FILE)
	$(ECHO_V)$(foreach pkg,$(LIST_PKGS),go test -bench=. -run="^$$" $(BENCH_FLAGS) $(pkg) | \
		tee -a $(BENCH_FILE);)

BASELINE_BENCH_FILE = .bench/old.txt
# Git diffs can be quote noisy, and contain all sorts of special characters, this just checks
# if there is anything in the output at all, which is what we want
GIT_DIFF = $(firstword $(shell git diff master))
.PHONY: benchbase
benchbase:
	$(ECHO_V)if [ -z "$(IGNORE_BASELINE_CHECK)" ] && [ -n "$(GIT_DIFF)" ]; then \
		echo "$(ERROR_STYLE)Can't record baseline with code changes off master." ; \
		echo "Check out master and try again$(COLOR_RESET)"; \
		exit 1; \
	fi

	@echo "$(LABEL_STYLE)Running baseline benchmark$(COLOR_RESET)"
	@echo
	$(ECHO_V)$(MAKE) bench BENCH_FILE=$(BASELINE_BENCH_FILE)

.PHONY: benchcmp
benchcmp:
	$(ECHO_V)which benchcmp >/dev/null || go get -u golang.org/x/tools/cmd/benchcmp
	$(ECHO_V)test -s $(BASELINE_BENCH_FILE) || \
		$(call die,Baseline benchmark file missing. Check out master and run \'make benchbase\')
	$(ECHO_V)test -s $(BENCH_FILE) || \
		$(call label,No current benchmark file. Will generate) ;\
	benchcmp $(BASELINE_BENCH_FILE) $(BENCH_FILE)

.PHONY: benchreset
benchreset:
	$(ECHO_V)rm -f $(BASELINE_BENCH_FILE)
	$(ECHO_V)rm -f $(BENCH_FILE)


include $(SUPPORT_FILES)/lint.mk
include $(SUPPORT_FILES)/licence.mk

.PHONY: gendoc
gendoc:
	$(ECHO_V)find . -name README.md \
		-not -path "./vendor/*" \
		-not -path "./node_modules/*" | \
		xargs -I% md-to-godoc -input=%

.PHONY: clean
clean:
	$(ECHO_V)rm -f $(COV_REPORT) $(COV_HTML) $(LINT_LOG)
	$(ECHO_V)find $(subst /...,,$(PKGS)) -name $(COVER_OUT) -delete
	$(ECHO_V)rm -rf examples/keyvalue/kv/

.PHONY: examples
examples: .bin/thriftrw .bin/thriftrw-plugin-yarpc
	@$(call label,Installing thriftrw and YARPC plugins)
	@echo
	$(ECHO_V)test -d vendor || $(MAKE) libdeps
	$(ECHO_V)which thriftrw-plugin-fx >/dev/null || go install ./modules/rpc/thriftrw-plugin-fx
	@$(call label,Generating example RPC bindings)
	@echo
	PATH=$(shell pwd)/.bin:$$PATH $(ECHO_)$(MAKE) -C examples/keyvalue kv/types.go ECHO_V=$(ECHO_V)

.bin/thriftrw:
	mkdir -p .bin
	./.build/build_vendored.sh .bin go.uber.org/thriftrw

.bin/thriftrw-plugin-yarpc:
	mkdir -p .bin
	./.build/build_vendored.sh .bin go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc
