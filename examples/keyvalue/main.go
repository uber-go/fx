package main

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/modules/rpc"
)

func main() {

	service := core.NewService(
		nil,
		rpc.ThriftModule("keyvalue", nil, rpc.CreateThriftServiceFunc(NewYarpcThriftHandler)),
	)

	service.Start(true)

}
