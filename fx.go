// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package fx

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/pkg/errors"

	"go.uber.org/dig"
	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

// Component is a dig constructor for something that's easily
// sharable in UberFx
type Component interface{}

// Module is a building block of UberFx
// TODO: Document and explain how is this different from Component?
// Something around roles and higher fidelity, maybe serving data
type Module interface {
	Name() string
	Constructor() Component
	Stop()
}

// Service foo
type Service struct {
	g                *dig.Graph
	modules          []Module
	moduleComponents []interface{}
	components       []Component
}

// New foo
func New(modules ...Module) *Service {
	s := &Service{
		g:       dig.New(),
		modules: modules,
	}

	// load config for module creation and include it in the graph
	cfg := config.DefaultLoader.Load()
	// TODO: need to pull latest dig for direct Interface injection fix
	s.g.MustRegister(func() config.Provider { return cfg })

	s.g.MustRegister(dispatcher)
	s.g.MustRegister(logger)

	// add a bunch of stuff
	for _, c := range modules {
		// TODO: everything is enabled right now
		co := c.Constructor()
		s.moduleComponents = append(s.moduleComponents, co)
		s.g.MustRegister(co)
	}

	return s
}

func dispatcher(cfg config.Provider, l *zap.Logger) *yarpc.Dispatcher {
	var c yarpcConfig
	// TODO: yarpc -> modules.yarpc
	err := cfg.Get("yarpc").Populate(&c)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%#v\n", c.Inbounds)
	inb, err := prepareInbounds(c.Inbounds, "noo")
	if err != nil {
		panic(err)
	}
	yc := yarpc.Config{
		Name:     "noo",
		Inbounds: inb,
	}
	return yarpc.NewDispatcher(yc)
}

func logger(cfg config.Provider) (*zap.Logger, error) {
	logConfig := ulog.Configuration{}
	logConfig.Configure(cfg.Get("logging"))
	l, err := logConfig.Build()
	return l, err
}

type yarpcConfig struct {
	transports transports
	Inbounds   []Inbound
}

type transports struct {
	inbounds []transport.Inbound
}

// Inbound is a union that configures how to configure a single inbound.
type Inbound struct {
	TChannel *Address
	HTTP     *Address
}

// Address is a struct that have a required port for tchannel/http transports.
// TODO(alsam) make it optional
type Address struct {
	Port int
}

func (i *Inbound) String() string {
	if i == nil {
		return ""
	}
	http := "none"
	if i.HTTP != nil {
		http = strconv.Itoa(i.HTTP.Port)
	}
	tchannel := "none"
	if i.TChannel != nil {
		tchannel = strconv.Itoa(i.TChannel.Port)
	}
	return fmt.Sprintf("Inbound:{HTTP: %s; TChannel: %s}", http, tchannel)
}

// Iterate over all inbounds and prepare corresponding transports
func prepareInbounds(inbounds []Inbound, serviceName string) (transportsIn []transport.Inbound, err error) {
	transportsIn = make([]transport.Inbound, 0, 2*len(inbounds))
	for _, in := range inbounds {
		if h := in.HTTP; h != nil {
			transportsIn = append(
				transportsIn,
				http.NewTransport().NewInbound(fmt.Sprintf(":%d", h.Port)))
		}

		if t := in.TChannel; t != nil {
			chn, err := tch.NewChannelTransport(
				tch.ServiceName(serviceName),
				tch.ListenAddr(fmt.Sprintf(":%d", t.Port)))

			if err != nil {
				return nil, errors.Wrap(err, "can't create tchannel transport")
			}

			transportsIn = append(transportsIn, chn.NewInbound())
		}
	}

	return transportsIn, nil
}

// WithComponents adds additional components to the service
func (s *Service) WithComponents(components ...Component) *Service {
	s.components = append(s.components, components...)

	// Add provided components to dig
	for _, c := range components {
		s.g.MustRegister(c)
	}

	return s
}

// Start foo
func (s *Service) Start() {
	// add a bunch of stuff
	// TODO: move to dig, perhaps #Call(constructor) function
	for _, c := range s.moduleComponents {
		ctype := reflect.TypeOf(c)
		switch ctype.Kind() {
		case reflect.Func:
			objType := ctype.Out(0)
			s.g.MustResolve(reflect.New(objType).Interface())
		}
	}

	// start the dispatcher
	var d *yarpc.Dispatcher
	s.g.MustResolve(&d)
	d.Start()

	// range over modules and start here
	fmt.Println("Stuff is started")
}

// Stop foo
func (s *Service) Stop() {
	for _, m := range s.modules {
		m.Stop()
	}
}
