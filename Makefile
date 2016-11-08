SHELL := /bin/bash
PROJECT_ROOT := go.uber.org/fx

SUPPORT_FILES := .build
include $(SUPPORT_FILES)/colors.mk
include $(SUPPORT_FILES)/deps.mk
include $(SUPPORT_FILES)/flags.mk
include $(SUPPORT_FILES)/verbosity.mk

.PHONY: all
all: lint test

COV_REPORT := overalls.coverprofile

# all .go files that don't exist in hidden directories
ALL_SRC := $(shell find . -name "*.go" | grep -v -e vendor \
	-e ".*/\..*" \
	-e "examples/keyvalue/.*" \
	-e ".*/_.*")

.PHONY: test
test: $(COV_REPORT)

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
	@echo "$(LABEL_STYLE)Generating example RPC bindings$(COLOR_RESET)"
	@echo
	$(ECHO_V)$(MAKE) -C examples/keyvalue/ kv/types.go ECHO_V=$(ECHO_V)
	@echo "$(LABEL_STYLE)Running tests$(COLOR_RESET)"
	@echo
	$(ECHO_V)$(OVERALLS) -project=$(PROJECT_ROOT) \
		-ignore "$(OVERALLS_IGNORE)" \
		-covermode=atomic \
		$(DEBUG_FLAG) -- \
		$(TEST_FLAGS) $(RACE) $(TEST_VERBOSITY_FLAG) | \
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

.PHONY: bench
BENCH ?= .
BENCH_FILE ?= .bench/new.txt
bench:
	$(ECHO_V)$(foreach pkg,$(BENCH_PKGS),go test -bench=. -run="^$$" $(BENCH_FLAGS) $(pkg) |\
		tee $(BENCH_FILE);)

.PHONY: benchbase
BASELINE_BENCH_FILE=.bench/old.txt
benchbase:
	$(ECHO_V)if [ -z "$(IGNORE_BASELINE_CHECK)" ] && [ -z "$(git diff master)" ]; then \
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
	$(ECHO_V)test -s $(BASELINE_BENCH_FILE) ||\
		$(error "Must have a baseline bench file. *Check out master* and run `make benchbase`")
	$(ECHO_V)test -s $(BENCH_FILE) || $(error "Must have a new benchmark file. Run `make bench`")
	$(ECHO_V)benchcmp $(BASELINE_BENCH_FILE) $(BENCH_FILE)

.PHONY: benchclean
benchclean:
	$(ECHO_V)if [ -a $(BASELINE_BENCH_FILE) ]; then rm $(BASELINE_BENCH_FILE); fi
	$(ECHO_V)if [ -a $(BENCH_FILE) ]; then rm $(BENCH_FILE); fi


include $(SUPPORT_FILES)/lint.mk
include $(SUPPORT_FILES)/licence.mk

.PHONY: gendoc
gendoc:
	$(ECHO_V)which md-to-godoc >/dev/null || go get -u github.com/sectioneight/md-to-godoc
	$(ECHO_V)find . -name README.md \
		-not -path "./vendor/*" \
		-not -path "./node_modules/*" | \
		xargs -I% md-to-godoc -input=%

.PHONY: clean
clean::
	$(ECHO_V)rm -f $(COV_REPORT) $(COV_HTML) $(LINT_LOG)
	$(ECHO_V)find $(subst /...,,$(PKGS)) -name $(COVER_OUT) -delete
	$(ECHO_V)rm -rf examples/keyvalue/kv/

