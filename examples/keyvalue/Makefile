.PHONY: deps
deps:
	$(ECHO_V)which thriftrw > /dev/null || go get -u go.uber.org/thriftrw
	$(ECHO_V)which thriftrw-plugin-yarpc > /dev/null || go get -u go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc

kv/types.go: kv.thrift deps
	$(ECHO_V)go generate

server/server: $(wildcard server/*.go) kv/types.go
	$(ECHO_V)go build -o $@ ./$(dir $@)

.PHONY: all
all: server/server

.DEFAULT_GOAL: all
