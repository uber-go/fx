# Directory containing the Makefile.
PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

export GOBIN ?= $(PROJECT_ROOT)/bin
export PATH := $(GOBIN):$(PATH)

FXLINT = $(GOBIN)/fxlint
MDOX = $(GOBIN)/mdox

MODULES = . ./tools ./docs ./internal/e2e

# 'make cover' should not run on docs by default.
# We run that separately explicitly on a specific platform.
COVER_MODULES ?= $(filter-out ./docs,$(MODULES))

.PHONY: all
all: build lint test

.PHONY: build
build:
	go build ./...

.PHONY: lint
lint: golangci-lint tidy-lint fx-lint docs-lint

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

.PHONY: tidy
tidy:
	@$(foreach dir,$(MODULES),(cd $(dir) && go mod tidy) &&) true

.PHONY: docs
docs:
	cd docs && yarn build

.PHONY: golangci-lint
golangci-lint:
	@$(foreach mod,$(MODULES), \
		(cd $(mod) && \
		echo "[lint] golangci-lint: $(mod)" && \
		golangci-lint run --path-prefix $(mod)) &&) true

.PHONY: tidy-lint
tidy-lint:
	@$(foreach mod,$(MODULES), \
		(cd $(mod) && \
		echo "[lint] tidy: $(mod)" && \
		go mod tidy && \
		git diff --exit-code -- go.mod go.sum) &&) true

.PHONY: fx-lint
fx-lint: $(FXLINT)
	@$(FXLINT) ./...

.PHONY: docs-lint
docs-lint: $(MDOX)
	@echo "Checking documentation"
	@make -C docs check

$(MDOX): tools/go.mod
	cd tools && go install github.com/bwplotka/mdox

$(FXLINT): tools/cmd/fxlint/main.go
	cd tools && go install go.uber.org/fx/tools/cmd/fxlint
