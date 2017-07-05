LINT_EXCLUDES = examples
# Create a pipeline filter for go vet/golint. Patterns specified in LINT_EXCLUDES are
# converted to a grep -v pipeline. If there are no filters, cat is used.
FILTER_LINT := $(if $(LINT_EXCLUDES), grep -v $(foreach file, $(LINT_EXCLUDES),-e $(file)),cat)

FILTER_LOG := grep -v "fx/examples" | grep -v "go.uber.org/fx/vendor/"

LINT_LOG := lint.log

_THIS_MAKEFILE := $(lastword $(MAKEFILE_LIST))
_THIS_DIR := $(dir $(_THIS_MAKEFILE))

ERRCHECK_FLAGS := -ignorepkg example -ignore "fmt.*,io:WriteString" -ignoretests

.PHONY: lint
lint:
	$(ECHO_V)rm -rf $(LINT_LOG)
	@echo "Installing test dependencies for vet..."
	$(ECHO_V)go test -i $(PKGS)
	@echo "Checking formatting..."
	$(ECHO_V)gofmt -d -s $(PKG_FILES) 2>&1 | tee $(LINT_LOG)
	@echo "Checking vet..."
	$(ECHO_V)go tool vet $(VET_RULES) $(ROOT_PKG_FILES) 2>&1 | $(FILTER_LINT) | tee -a $(LINT_LOG)
	$(ECHO_V)go tool vet $(VET_RULES) $(PKG_NAMES) 2>&1 | $(FILTER_LINT) | tee -a $(LINT_LOG)
	@echo "Checking lint..."
	$(ECHO_V)$(foreach dir,$(PKGS),golint $(dir) 2>&1 | $(FILTER_LINT) | tee -a $(LINT_LOG);)
	@echo "Checking unchecked errors..."
	$(ECHO_V)errcheck $(ERRCHECK_FLAGS) $(PKGS) 2>&1 | $(FILTER_LINT) | tee -a $(LINT_LOG)
	@echo "Checking for unresolved FIXMEs..."
	$(ECHO_V)git grep -i fixme | grep -v -e vendor -e $(_THIS_MAKEFILE) -e CONTRIBUTING.md | tee -a $(LINT_LOG)
	@echo "Checking for license headers..."
	$(ECHO_V)DRY_RUN=1 $(_THIS_DIR)/license.sh | tee -a $(LINT_LOG)
	@echo "Checking for imports of log package"
	$(ECHO_V)go list -f '{{ .ImportPath }}: {{ .Imports }}' $(shell glide nv) | grep -e "\blog\b" | $(FILTER_LOG) | tee -a $(LINT_LOG)
	@echo "Ensuring generated doc.go are up to date"
	$(ECHO_V)$(MAKE) gendoc
	$(ECHO_V)[ -z "$(shell git status --porcelain | grep '\bdoc.go$$')" ] || echo "Commit updated doc.go changes" | tee -a $(LINT_LOG)
	$(ECHO_V)[ ! -s $(LINT_LOG) ]
