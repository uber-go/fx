.DEFAULT_GOAL := all

kv/types.go: deps kv.thrift
	$(ECHO_V)go generate
	$(ECHO_V)go get go.uber.org/update-license
	$(ECHO_V)update-license $(shell find kv -name '*.go')


server/server: $(wildcard server/*.go) kv/types.go
	$(ECHO_V)go build -o $@ ./$(dir $@)

.PHONY: all
all: server/server

.PHONY: deps
deps:
	@echo "Installing thriftrw..."
	$(ECHO_V)go install ../../vendor/go.uber.org/thriftrw

	@echo "Installing thriftrw-plugin-yarpc..."
	$(ECHO_V)go install ../../vendor/go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc

	@echo "Installing thriftrw-plugin-thriftsync..."
	$(ECHO_V)go install ../../modules/yarpc/thriftrw-plugin-thriftsync

.PHONY: thriftsync
thriftsync: deps
	$(ECHO_V)thriftrw --pkg-prefix=$(PROJECT_ROOT) --plugin="thriftsync --handler-package=main" kv.thrift
