package main

// import (
// 	"github.com/uber-go/uberfx/core"
// 	"github.com/uber-go/uberfx/modules/kafka"
// 	"golang.org/x/net/context"
// )

// // Create the handler instance for the given topic
// func CreateMyTopicHandler(topic string, service *core.Service) kafka.HandlerCreateFunc {
// 	return func() (kakfa.Handler, error) {
// 		instance := &MyTopicHandler{}
// 		return instance.MyTopic, nil
// 	}
// }

// type MyTopicHandler struct {
// }

// // MyTopic handles messages from topic 'my_topic'
// func (h *MyTopicHandler) MyTopic(ctx context.Context, message consumer.Message) error {
// 	log.Infof("I got a message on topic %q", message.Topic())
// 	return nil
// }
