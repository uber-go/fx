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
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
)

// ShutdownSignal is a signal that caused the application to exit.
type ShutdownSignal struct {
	Signal os.Signal
}

// String will render a ShutdownSignal type as a string suitable for printing.
func (sig ShutdownSignal) String() string {
	return fmt.Sprintf("%v", sig.Signal)
}

func newSignalReceivers() signalReceivers {
	return signalReceivers{notify: signal.Notify}
}

type signalReceivers struct {
	m           sync.Mutex
	relayCloser func(context.Context) error
	last        *ShutdownSignal
	done        []chan os.Signal
	notify      func(c chan<- os.Signal, sig ...os.Signal)
}

func (recv *signalReceivers) StartSignalRelayer() error {
	recv.m.Lock()
	defer recv.m.Unlock()

	if recv.relayCloser != nil {
		return errors.New("signal relay is still running")
	}

	waitForClose, closeRelayer := context.WithCancel(context.Background())
	relayHasStopped, signalRelayIsStopped := context.WithCancel(context.Background())

	recv.relayCloser = func(ctx context.Context) error {
		closeRelayer()
		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"waiting for signal relay to close: %w",
				ctx.Err(),
			)
		case <-relayHasStopped.Done():
			return nil
		}
	}

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, recv.signalsToRelay()...)
		defer signalRelayIsStopped()
		select {
		case <-waitForClose.Done():
			return
		case signal := <-ch:
			recv.Broadcast(ShutdownSignal{
				Signal: signal,
			})
			recv.m.Lock()
			defer recv.m.Unlock()
			recv.relayCloser = nil
			return
		}
	}()

	return nil
}

func (recv *signalReceivers) StopSignalRelayer(ctx context.Context) error {
	recv.m.Lock()
	defer recv.m.Lock()

	if recv.relayCloser == nil {
		return errors.New("signal relayer is not started")
	}

	return recv.relayCloser(ctx)
}

func (recv *signalReceivers) signalsToRelay() []os.Signal {
	return []os.Signal{os.Interrupt, _sigINT, _sigTERM}
}

func (recv *signalReceivers) Done() <-chan os.Signal {
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

	recv.notify(ch, os.Interrupt, _sigINT, _sigTERM)
	recv.done = append(recv.done, ch)
	return ch
}

func (recv *signalReceivers) Broadcast(signal ShutdownSignal) error {
	recv.m.Lock()
	defer recv.m.Unlock()

	recv.last = &signal

	channels, unsent := recv.broadcastDone(signal)

	if unsent != 0 {
		return &unsentSignalError{
			Signal: signal,
			Total:  channels,
			Unsent: unsent,
		}
	}

	return nil
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
