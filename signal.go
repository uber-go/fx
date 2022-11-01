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
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package fx

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
)

// ShutdownSignal represents a operating system process signal.
type ShutdownSignal struct {
	OS os.Signal
}

// String will render a Signal type as a string suitable for printing.
func (sig ShutdownSignal) String() string {
	return fmt.Sprintf("%v", sig.OS)
}

type signalReceivers struct {
	m     sync.Mutex
	last  *ShutdownSignal
	done []chan os.Signal
}

func (recv *signalReceivers) Done() chan os.Signal {
	recv.m.Lock()
	defer recv.m.Unlock()

	ch := make(chan os.Signal, 1)

	// if we had received a signal prior to the call of done, send it's
	// os.Signal to the new channel.
	if recv.last != nil {
		ch <- recv.last.OS
	}

	signal.Notify(ch, os.Interrupt, _sigINT, _sigTERM)
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
			Signal:   signal,
			Channels: channels,
			Unsent:   unsent,
		}
	}

	return nil
}

func (recv *signalReceivers) broadcastDone(signal ShutdownSignal) (int, int) {
	var (
		receivers int = len(recv.done)
		unsent    int
	)

	for _, reader := range recv.done {
		select {
		case reader <- signal.OS:
		default:
			unsent++
		}
	}

	return receivers, unsent
}

type unsentSignalError struct {
	Signal   ShutdownSignal
	Unsent   int
	Channels int
}

func (err *unsentSignalError) Error() string {
	return fmt.Sprintf(
		"send %v signal: %v/%v channels are blocked",
		err.Signal,
		err.Unsent,
		err.Channels,
	)
}
