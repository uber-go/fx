export GOBIN ?= $(shell pwd)/bin

GOLINT = $(GOBIN)/golint
STATICCHECK = $(GOBIN)/staticcheck
FXLINT = $(GOBIN)/fxlint

GO_FILES := $(shell \
	find . '(' -path '*/.*' -o -path './vendor' -o -path '*/testdata/*' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

.PHONY: build
build:
	go build ./...

.PHONY: install
install:
	go mod download

.PHONY: test
test:
	go test -race ./...

.PHONY: cover
cover:
	go test -race -coverprofile=cover.out -coverpkg=./... ./...
	go tool cover -html=cover.out -o cover.html

$(GOLINT): tools/go.mod
	cd tools && go install golang.org/x/lint/golint

$(STATICCHECK): tools/go.mod
	cd tools && go install honnef.co/go/tools/cmd/staticcheck

$(FXLINT): $(shell find tools -name '*.go')
	cd tools && go install go.uber.org/fx/tools/cmd/fxlint

.PHONY: lint
lint: $(GOLINT) $(STATICCHECK) $(FXLINT)
	@rm -rf lint.log
	@echo "Checking formatting..."
	@gofmt -d -s $(GO_FILES) 2>&1 | tee lint.log
	@echo "Checking vet..."
	@go vet ./... 2>&1 | tee -a lint.log
	@echo "Checking lint..."
	@$(GOLINT) ./... | tee -a lint.log
	@echo "Checking staticcheck..."
	@$(STATICCHECK) ./... | tee -a lint.log
	@echo "Checking fxlint..."
	@$(FXLINT) ./... | tee -a lint.log
	@echo "Checking for unresolved FIXMEs..."
	@git grep -i fixme | grep -v -e vendor -e Makefile -e .md | tee -a lint.log
	@echo "Checking for license headers..."
	@./checklicense.sh | tee -a lint.log
	@[ ! -s lint.log ]
