PKGS ?= $(shell glide novendor | grep -v examples)
PKG_FILES ?= *.go

# The linting tools evolve with each Go version, so run them only on the latest
# stable release.
GO_VERSION := $(shell go version | cut -d " " -f 3)
GO_MINOR_VERSION := $(word 2,$(subst ., ,$(GO_VERSION)))
LINTABLE_MINOR_VERSIONS := 8
ifneq ($(filter $(LINTABLE_MINOR_VERSIONS),$(GO_MINOR_VERSION)),)
SHOULD_LINT := true
endif

.PHONY: install
install:
	glide --version || go get github.com/Masterminds/glide
	glide install

.PHONY: test
test:
	@.build/test.sh

.PHONY: license
license:
	$(ECHO_V).build/check_license.sh

.PHONY: ci
ci: SHELL := /bin/bash
ci: test
	bash <(curl -s https://codecov.io/bash)

.PHONY: lint
lint:
ifdef SHOULD_LINT
	@rm -rf lint.log
	@echo "Checking formatting..."
	@gofmt -d -s $(PKG_FILES) 2>&1 | tee lint.log
	@echo "Installing test dependencies for vet..."
	@go test -i $(PKGS)
	@echo "Checking vet..."
	@$(foreach dir,$(PKG_FILES),go tool vet $(VET_RULES) $(dir) 2>&1 | tee -a lint.log;)
	@echo "Checking lint..."
	@$(foreach dir,$(PKGS),golint $(dir) 2>&1 | tee -a lint.log;)
	@echo "Checking for unresolved FIXMEs..."
	@git grep -i fixme | grep -v -e vendor -e Makefile -e .md | tee -a lint.log
	@echo "Checking for license headers..."
	@DRY_RUN=1 .build/check_license.sh | tee -a lint.log
	@$(MAKE) gendoc
	@[ ! -s lint.log ]
else
	@echo "Skipping linters on" $(GO_VERSION)
endif

.PHONY: gendoc
gendoc:
	@echo "Generating doc.go from README.md..."
	@find . -name README.md -not -path "./vendor/*" | xargs -I% md-to-godoc -input=%
	@# doc.go gets regenerated, so refresh its license
	@update-license doc.go
