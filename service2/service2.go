package service2

import (
	"log"
	"reflect"

	"go.uber.org/dig"
	"go.uber.org/fx/config"
)

// Service foo
type Service struct {
	g  *dig.Graph
	cs []interface{}
}

// Start foo
func (s *Service) Start() {
	// add a bunch of stuff
	// TODO: move to dig, perhaps #Call(constructor) function
	for _, c := range s.cs {
		ctype := reflect.TypeOf(c)
		switch ctype.Kind() {
		case reflect.Func:
			objType := ctype.Out(0)
			s.g.MustResolve(reflect.New(objType).Interface())
		}
	}
}

// Stop foo
func (s *Service) Stop() {
	// close all dig stuff
	log.Println("Stopping...")
}

// New foo
func New(constructors ...interface{}) *Service {
	s := &Service{
		g:  dig.New(),
		cs: constructors,
	}

	s.g.MustRegister(config.DefaultLoader.Load)

	// add a bunch of stuff
	for _, c := range constructors {
		s.g.MustRegister(c)
	}

	return s
}
