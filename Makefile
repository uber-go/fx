SHELL := /bin/bash
PROJECT_ROOT := github.com/uber-go/uberfx

SUPPORT_FILES := .build
include $(SUPPORT_FILES)/flags.mk
include $(SUPPORT_FILES)/verbosity.mk
include $(SUPPORT_FILES)/deps.mk

.PHONY: all
all: lint test

COV_REPORT := overalls.coverprofile

# all .go files that don't exist in hidden directories
ALL_SRC := $(shell find . -name "*.go" | grep -v -e vendor \
  -e ".*/\..*" \
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

$(COV_REPORT): $(PKG_FILES) $(ALL_SRC)
	$(ECHO_V)$(OVERALLS) -project=$(PROJECT_ROOT) \
		-ignore "$(OVERALLS_IGNORE)" \
		-covermode=atomic \
		$(DEBUG_FLAG) -- \
		$(TEST_FLAGS) $(RACE) $(TEST_VERBOSITY_FLAG) | \
		grep -v "No Go Test files" | \
		$(_FILTER_OVERALLS)
	$(ECHO_V)$(GOCOV) convert $@ | $(GOCOV) report

COV_HTML := coverage.html

$(COV_HTML): $(COV_REPORT)
	$(ECHO_V)$(GOCOV) convert $< | gocov-html > $@

.PHONY: coveralls
coveralls: $(COV_REPORT)
	$(ECHO_V)goveralls -service=travis-ci .

.PHONY: bench
BENCH ?= .
bench:
	$(ECHO_V)$(foreach pkg,$(PKGS),go test -bench=$(BENCH) -run="^$$" $(BENCH_FLAGS) $(pkg);)

include $(SUPPORT_FILES)/lint.mk
include $(SUPPORT_FILES)/licence.mk

.PHONY: clean
clean::
	@rm -f $(COV_REPORT) $(COV_HTML) $(LINT_LOG)

