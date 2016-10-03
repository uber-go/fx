LINT_EXCLUDES = examples
# Create a pipeline filter for go vet/golint. Patterns specified in LINT_EXCLUDES are
# converted to a grep -v pipeline. If there are no filters, cat is used.
FILTER_LINT := $(if $(LINT_EXCLUDES), grep -v $(foreach file, $(LINT_EXCLUDES),-e $(file)),cat)

LINT_LOG := lint.log

.PHONY: lint
lint:
ifdef SHOULD_LINT
	$(ECHO_V)rm -rf $(LINT_LOG)
	@echo "Checking formatting..."
	$(ECHO_V)gofmt -d -s $(PKG_FILES) 2>&1 | tee $(LINT_LOG)
	@echo "Installing test dependencies for vet..."
	$(ECHO_V)go test -i $(PKGS)
	@echo "Checking vet..."
	$(ECHO_V)$(foreach dir,$(PKG_FILES),go tool vet $(VET_RULES) $(dir) 2>&1 | $(FILTER_LINT) | tee -a $(LINT_LOG);)
	@echo "Checking lint..."
	$(ECHO_V)$(foreach dir,$(PKGS),golint $(dir) 2>&1 | $(FILTER_LINT) | tee -a $(LINT_LOG);)
	@echo "Checking for unresolved FIXMEs..."
	$(ECHO_V)git grep -i fixme | grep -v -e vendor -e $(lastword $(MAKEFILE_LIST)) | tee -a $(LINT_LOG)
	@echo "Checking for license headers..."
	$(ECHO_V).bin/check_license.sh | tee -a $(LINT_LOG)
	$(ECHO_V)[ ! -s $(LINT_LOG) ]
else
	@echo "Skipping linters on" $(GO_VERSION)
endif
