.PHONY: deps
deps:
	@echo "Installing thriftrw..."
	$(ECHO_V)go install ../../vendor/go.uber.org/thriftrw
	@echo "Installing thriftrw-plugin-yarpc..."
	$(ECHO_V)go install ../../vendor/go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc

kv/types.go: kv.thrift deps
	$(ECHO_V)go generate

server/server: $(wildcard server/*.go) kv/types.go
	$(ECHO_V)go build -o $@ ./$(dir $@)

.PHONY: all
all: server/server

.DEFAULT_GOAL: all
