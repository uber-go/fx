package main

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/modules"
	"github.com/uber-go/uberfx/modules/rpc"
)

func main() {

	service := core.New(
		&MyService{},
		core.WithModules(

			// Create a YARPC module that exposes endpoints
			rpc.ThriftModule(
				"rpc",
				nil,
				rpc.CreateThriftServiceFunc(NewYarpcThriftHandler),
				modules.WithRoles("service")),

			// // Create a kakfa module that works on a topic
			// //
			// messaging.Kafka(
			// 	NewMyTopicHandler,
			// 	messaging.WithTopic("my_topic"),
			// 	modules.WithRoles("worker"),
			// ),
		),
	)
	service.Start(true)

}
