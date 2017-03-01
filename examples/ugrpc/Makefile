.PHONY: all
all: proto

.PHONY: proto
proto:
	go install ../../vendor/github.com/golang/protobuf/protoc-gen-go
	protoc --go_out=plugins=grpc:. kv/kv.proto

.PHONY: server
server:
	cd ./server && go run *.go

.PHONY: client
client:
	cd ./client && go run *.go set one hello
	cd ./client && go run *.go get one
