.PHONY: deps
deps:
	which thriftrw > /dev/null || go get -u go.uber.org/thriftrw
	which thriftrw-plugin-yarpc > /dev/null || go get -u go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc

kv/types.go: kv.thrift deps
	go generate

server/server: $(wildcard server/*.go) kv/types.go
	go build -o $@ ./$(dir $@)

.PHONY: all
all: server/server

.DEFAULT_GOAL: all
