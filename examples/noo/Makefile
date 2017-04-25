.PHONY: yab
yab:
	yab --service noo -p :1234 -m KeyValue::getValue -t kv.thrift -r '{"key":"key1"}'
	yab --service noo -p :1234 -m KeyValue::setValue -t kv.thrift -r '{"key":"key1","value":"value1"}'
	yab --service noo -p :1234 -m KeyValue::getValue -t kv.thrift -r '{"key":"key1"}'

.PHONY: run
run:
	go run main.go

