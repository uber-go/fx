SHELL := /bin/bash
PROJECT_ROOT := go.uber.org/fx

SUPPORT_FILES := .build
include $(SUPPORT_FILES)/colors.mk
include $(SUPPORT_FILES)/deps.mk
include $(SUPPORT_FILES)/flags.mk
include $(SUPPORT_FILES)/verbosity.mk
include $(SUPPORT_FILES)/lint.mk

.PHONY: all
all: lint test
.DEFAULT_GOAL := all

COV_REPORT := overalls.coverprofile

DOCKER_GO_VERSION ?= 1.8
DOCKER_IMAGE := uber/fx-$(DOCKER_GO_VERSION)
DOCKERFILE := Dockerfile.$(DOCKER_GO_VERSION)
DOCKER_CACHE_DIR ?= .cache/docker
DOCKER_CACHE_FILE := $(DOCKER_CACHE_DIR)/uber-fx-$(DOCKER_GO_VERSION)
DOCKER_FLAGS := -e V -e COVERMODE -e RACE -e CI_TEST_CMD -e TRAVIS_JOB_ID -e TRAVIS_PULL_REQUEST

COVERMODE ?= set
CI_TEST_CMD ?= test
ifeq ($(CI_TEST_CMD),coveralls)
COVERMODE := atomic
endif

# all .go files that don't exist in hidden directories
ALL_SRC := $(shell find . -name "*.go" | grep -v -e vendor \
	-e ".*/\..*" \
	-e "examples/.*" \
	-e ".*/_.*")

TEST_TIMEOUT := "-timeout=10s"

.PHONY: test
test: $(COV_REPORT)

TEST_IGNORES = vendor .git
COVER_IGNORES = $(TEST_IGNORES) examples testutils

comma := ,
null :=
space := $(null) #
OVERALLS_IGNORE = $(subst $(space),$(comma),$(strip $(COVER_IGNORES)))

ifeq ($(V),0)
_FILTER_OVERALLS = grep -v "^Test args"
else
_FILTER_OVERALLS = grep -v "^Processing:"
endif

COV_TXT = coverage.txt
ifeq ($(COVER),1)
_OUTPUT_COVERAGE = cat -
else
_OUTPUT_COVERAGE = cat - > $(COV_TXT)
endif

# This is the default for overalls
COVER_OUT := profile.coverprofile

$(COV_REPORT): $(PKG_FILES) $(ALL_SRC)
	@$(call label,Cleaning old profile)
	$(ECHO_V)rm -f $(COV_REPORT)

	@$(call label,Running tests)
	$(ECHO_V)RICHGO_FORCE_COLOR=1 $(OVERALLS) \
		-project=$(PROJECT_ROOT) \
		-go-binary=richgo \
		-ignore "$(OVERALLS_IGNORE)" \
		-covermode=$(COVERMODE) \
		$(DEBUG_FLAG) -- \
		$(TEST_FLAGS) $(RACE) $(TEST_TIMEOUT) $(TEST_VERBOSITY_FLAG) | \
		grep -v "No Go Test files" | \
		$(_FILTER_OVERALLS)

	@$(call label,Generating coverage report)
	$(ECHO_V)rm -f $(COV_TXT)
	$(ECHO_V)if [ -a $(COV_REPORT) ]; then \
		$(call label,Tests succeeded) | $(_OUTPUT_COVERAGE) ; \
		$(GOCOV) convert $@ | $(GOCOV) report | $(_OUTPUT_COVERAGE) ; \
	else \
		$(call die,Tests failed); \
	fi;

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
	$(ECHO_V)$(foreach pkg,$(LIST_PKGS),richgo test -bench=. -run="^$$" $(BENCH_FLAGS) $(pkg) | \
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


.PHONY: dockerbuild
dockerbuild:
	docker build -t $(DOCKER_IMAGE) -f $(DOCKERFILE) .

.PHONY: dockertest
dockertest: dockerbuild
	docker run $(DOCKER_FLAGS) $(DOCKER_IMAGE) make test

.PHONY: dockerci
dockerci: dockerbuild
	docker run $(DOCKER_FLAGS) $(DOCKER_IMAGE) make ci

.PHONY: dockerload
dockerload:
	if [ -f $(DOCKER_CACHE_FILE) ]; then gunzip -c $(DOCKER_CACHE_FILE) | docker load; fi

.PHONY: dockersave
dockersave:
	mkdir -p $(shell dirname $(DOCKER_CACHE_FILE))
	docker save $(shell docker history -q $(DOCKER_IMAGE) | grep -v '<missing>') | gzip > $(DOCKER_CACHE_FILE)

.PHONY: ci
ci: lint examples $(CI_TEST_CMD)

.PHONY: gendoc
gendoc:
	$(ECHO_V)find . -name README.md \
		-not -path "./vendor/*" \
		-not -path "./node_modules/*" | \
		xargs -I% md-to-godoc -input=%

.PHONY: genexamples
genexamples:
	@$(call label,Building examples)
	@echo
	$(ECHO_V)$(MAKE) -C examples/keyvalue ECHO_V=$(ECHO_V)

.PHONY: license
license:
	$(ECHO_V)./.build/license.sh

.PHONY: generate
generate: gendoc genexamples license

.PHONY: clean
clean:
	$(ECHO_V)rm -f $(COV_REPORT) $(COV_HTML) $(LINT_LOG) $(COV_TXT)
	$(ECHO_V)find $(subst /...,,$(PKGS)) -name $(COVER_OUT) -delete
	$(ECHO_V)rm -rf .bin

.PHONY: examples
examples:
	$(ECHO_V)go test ./examples/simple
	$(ECHO_V)go test ./examples/dig

.PHONY: vendor
vendor:
	$(ECHO_V)test -d vendor || $(MAKE) libdeps
