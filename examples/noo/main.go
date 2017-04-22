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

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/config"
	"go.uber.org/fx/ulog"
	"go.uber.org/zap"
)

func main() {
	svc := fx.New(
		NewT(),
	).WithComponents(
		FxZapNew,
	)
	svc.Start()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Println(<-c)
	svc.Stop()
}

// FxZapNew is a component constructor thing for zap
func FxZapNew(cfg config.Provider) (*zap.Logger, error) {
	fmt.Println("New zap was called")

	logConfig := ulog.Configuration{}
	logConfig.Configure(cfg.Get("logging"))
	l, err := logConfig.Build()
	return l, err
}

type ticker struct {
	t *time.Ticker
}

// NewT foo
func NewT() *ticker {
	return &ticker{}
}

func (t *ticker) Name() string { return "ticker" }
func (t *ticker) Constructor() fx.Component {
	return func(l *zap.Logger) *time.Ticker {
		fmt.Println("new ticker was called")

		ticker := time.NewTicker(time.Second * 1)
		go func() {
			for range ticker.C {
				l.Info("I'm alive")
			}
		}()
		t.t = ticker
		return ticker
	}
}
func (t *ticker) Stop() {
	fmt.Println("gnight")
	if t.t != nil {
		t.t.Stop()
	}
}
