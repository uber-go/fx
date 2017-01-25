.DEFAULT_GOAL := all

kv/types.go: kv.thrift
	$(ECHO_V)go generate

server/server: $(wildcard server/*.go) kv/types.go
	$(ECHO_V)go build -o $@ ./$(dir $@)

.PHONY: all
all: server/server

.PHONY: deps
deps:
	$(ECHO_V)go get -v go.uber.org/thriftrw
	$(ECHO_V)go get -v go.uber.org/yarpc/encoding/thrift/thriftrw-plugin-yarpc
