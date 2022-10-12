export GOBIN ?= $(shell pwd)/bin

GOLINT = $(GOBIN)/golint
STATICCHECK = $(GOBIN)/staticcheck
FXLINT = $(GOBIN)/fxlint
MDOX = $(GOBIN)/mdox

GO_FILES = $(shell \
	find . '(' -path '*/.*' -o -path './vendor' -o -path '*/testdata/*' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

MODULES = . ./tools ./docs

# 'make cover' should not run on docs by default.
# We run that separately explicitly on a specific platform.
COVER_MODULES ?= $(filter-out ./docs,$(MODULES))

.PHONY: build
build:
	go build ./...

.PHONY: install
install:
	go mod download

.PHONY: test
test:
	@$(foreach dir,$(MODULES),(cd $(dir) && go test -race ./...) &&) true

.PHONY: cover
cover:
	@$(foreach dir,$(COVER_MODULES), \
		(cd $(dir) && \
		echo "[cover] $(dir)" && \
		go test -race -coverprofile=cover.out -coverpkg=./... ./... && \
		go tool cover -html=cover.out -o cover.html) &&) true

$(GOLINT): tools/go.mod
	cd tools && go install golang.org/x/lint/golint

$(STATICCHECK): tools/go.mod
	cd tools && go install honnef.co/go/tools/cmd/staticcheck

$(MDOX): tools/go.mod
	cd tools && go install github.com/bwplotka/mdox

$(FXLINT): tools/cmd/fxlint/main.go
	cd tools && go install go.uber.org/fx/tools/cmd/fxlint

.PHONY: lint
lint: $(GOLINT) $(STATICCHECK) $(FXLINT) docs-check
	@rm -rf lint.log
	@echo "Checking formatting..."
	@gofmt -d -s $(GO_FILES) 2>&1 | tee lint.log
	@echo "Checking vet..."
	@$(foreach dir,$(MODULES),(cd $(dir) && go vet ./... 2>&1) &&) true | tee -a lint.log
	@echo "Checking lint..."
	@$(foreach dir,$(MODULES),(cd $(dir) && $(GOLINT) ./... 2>&1) &&) true | tee -a lint.log
	@echo "Checking staticcheck..."
	@$(foreach dir,$(MODULES),(cd $(dir) && $(STATICCHECK) ./... 2>&1) &&) true | tee -a lint.log
	@echo "Checking fxlint..."
	@$(FXLINT) ./... | tee -a lint.log
	@echo "Checking for unresolved FIXMEs..."
	@git grep -i fixme | grep -v -e vendor -e Makefile -e .md | tee -a lint.log
	@echo "Checking for license headers..."
	@./checklicense.sh | tee -a lint.log
	@[ ! -s lint.log ]
	@echo "Checking 'go mod tidy'..."
	@make tidy
	@if ! git diff --quiet; then \
		echo "'go mod tidy' resulted in changes or working tree is dirty:"; \
		git --no-pager diff; \
	fi

.PHONY: docs
docs:
	cd docs && yarn build

.PHONY: docs-check
docs-check: $(MDOX)
	@echo "Checking documentation"
	@make -C docs check | tee -a lint.log

.PHONY: tidy
tidy:
	@$(foreach dir,$(MODULES),(cd $(dir) && go mod tidy) &&) true
