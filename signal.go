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

type Signal struct {
	OS   os.Signal
	Code int
}

func (sig Signal) String() string {
	return fmt.Sprintf("%v (with exit code %v)", sig.OS, sig.Code)
}

type signalReceivers struct {
	last       *Signal
	lastLock   sync.RWMutex
	done       []chan<- os.Signal
	doneLock   sync.RWMutex
	signal     []chan<- Signal
	signalLock sync.RWMutex
}

func (recv *signalReceivers) Done() chan<- os.Signal {
	recv.doneLock.Lock()
	defer recv.doneLock.Unlock()
	recv.lastLock.RLock()
	defer recv.lastLock.RUnlock()

	ch := make(chan<- os.Signal, 1)

	// if we had received a signal prior to the call of Done, send it's os.Signal
	// to the new channel.
	if recv.last != nil {
		ch <- recv.last.OS
	}

	signal.Notify(ch, os.Interrupt, _sigINT, _sigTERM)
	recv.done = append(recv.done, ch)
	return ch
}

type unsentSignalError struct {
	Signal   Signal
	Unsent   int
	Channels int
}

func (err unsentSignalError) Error() string {
	return fmt.Sprintf(
		"send %v signal: %v/%v channels are blocked",
		err.Signal,
		err.Unsent,
		err.Channels,
	)
}

func (recv signalReceivers) broadcastDone(signal Signal) (total, unsent int) {
	recv.doneLock.RLock()
	defer recv.doneLock.RUnlock()

	total = len(recv.done)

	for _, reader := range recv.done {
		select {
		case reader <- signal.OS:
		default:
			unsent++
		}
	}

	return
}

func (recv *signalReceivers) Broadcast(signal Signal) (err error) {
	recv.lastLock.Lock()
	recv.last = &signal
	recv.lastLock.Unlock()

	dones, doneUnsent := recv.broadcastDone(signal)

	unsent := doneUnsent
	channels := dones

	if unsent != 0 {
		err = &unsentSignalError{
			Signal:   signal,
			Channels: channels,
			Unsent:   unsent,
		}
	}

	return
}
