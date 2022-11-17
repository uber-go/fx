// Copyright (c) 2022 Uber Technologies, Inc.
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
// FITNESS FOR A PARTICULAR PURPSignalE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package fx

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
)

// ShutdownSignal represents a signal to be written to Wait or Done.
// Should a user call the Shutdown method via the Shutdowner interface with
// a provided ExitCode, that exit code will be populated in the ExitCode field.
//
// Should the application receive an operating system signal,
// the Signal field will be populated with the received os.Signal.
type ShutdownSignal struct {
	Signal   os.Signal
	ExitCode int
}

// String will render a ShutdownSignal type as a string suitable for printing.
func (sig ShutdownSignal) String() string {
	return fmt.Sprintf("%v", sig.Signal)
}

func newSignalReceivers() signalReceivers {
	return signalReceivers{
		notify:  signal.Notify,
		signals: make(chan os.Signal, 1),
	}
}

type signalReceivers struct {
	m        sync.Mutex
	signals  chan os.Signal
	shutdown chan struct{}
	finished chan struct{}
	notify   func(c chan<- os.Signal, sig ...os.Signal)

	last *ShutdownSignal
	done []chan os.Signal
	wait []chan ShutdownSignal
}

func (recv *signalReceivers) relayer(ctx context.Context) {
	defer func() {
	recv.finished <- struct{}{}
	}()

	select {
	case <-recv.shutdown:
		return
	case <-ctx.Done():
		return
	case signal := <-recv.signals:
		recv.Broadcast(ShutdownSignal{
			Signal: signal,
		})
	}
}

func (recv *signalReceivers) running() bool {
	return recv.shutdown != nil && recv.finished != nil
}

func (recv *signalReceivers) Start(ctx context.Context) {
	recv.m.Lock()
	defer recv.m.Unlock()

	recv.last = nil
	recv.finished = make(chan struct{}, 1)
	recv.shutdown = make(chan struct{}, 1)
	recv.notify(recv.signals, os.Interrupt, _sigINT, _sigTERM)
	go recv.relayer(ctx)
}

func (recv *signalReceivers) Stop(ctx context.Context) error {
	recv.m.Lock()
	defer recv.m.Unlock()

	// if the relayer is not running; return nil error
	if !recv.running() {
		return nil
	}

	recv.shutdown <- struct{}{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-recv.finished:
			close(recv.shutdown)
			close(recv.finished)
			recv.shutdown = nil
			recv.finished = nil
			return nil
		}

	}
}

func (recv *signalReceivers) Done() chan os.Signal {
	recv.m.Lock()
	defer recv.m.Unlock()

	ch := make(chan os.Signal, 1)

	// If we had received a signal prior to the call of done, send it's
	// os.Signal to the new channel.
	// However we still want to have the operating system notify signals to this
	// channel should the application receive another.
	if recv.last != nil {
		ch <- recv.last.Signal
	}

	recv.done = append(recv.done, ch)
	return ch
}

func (recv *signalReceivers) Wait() chan ShutdownSignal {
	recv.m.Lock()
	defer recv.m.Unlock()

	ch := make(chan ShutdownSignal, 1)

	if recv.last != nil {
		ch <- *recv.last
	}

	recv.wait = append(recv.wait, ch)
	return ch
}

func (recv *signalReceivers) Broadcast(signal ShutdownSignal) error {
	recv.m.Lock()
	defer recv.m.Unlock()

	recv.last = &signal

	channels, unsent := recv.broadcast(
		signal,
		recv.broadcastDone,
		recv.broadcastWait,
	)

	if unsent != 0 {
		return &unsentSignalError{
			Signal: signal,
			Total:  channels,
			Unsent: unsent,
		}
	}

	return nil
}

func (recv *signalReceivers) broadcast(
	signal ShutdownSignal,
	anchors ...func(ShutdownSignal) (int, int),
) (int, int) {
	var channels, unsent int

	for _, anchor := range anchors {
		c, u := anchor(signal)
		channels += c
		unsent += u
	}

	return channels, unsent
}

func (recv *signalReceivers) broadcastDone(signal ShutdownSignal) (int, int) {
	var unsent int

	for _, reader := range recv.done {
		select {
		case reader <- signal.Signal:
		default:
			unsent++
		}
	}

	return len(recv.done), unsent
}

func (recv *signalReceivers) broadcastWait(signal ShutdownSignal) (int, int) {
	var unsent int

	for _, reader := range recv.wait {
		select {
		case reader <- signal:
		default:
			unsent++
		}
	}

	return len(recv.wait), unsent
}

type unsentSignalError struct {
	Signal ShutdownSignal
	Unsent int
	Total  int
}

func (err *unsentSignalError) Error() string {
	return fmt.Sprintf(
		"send %v signal: %v/%v channels are blocked",
		err.Signal,
		err.Unsent,
		err.Total,
	)
}
