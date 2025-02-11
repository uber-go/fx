// Copyright (c) 2024 Uber Technologies, Inc.
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
	"sync"
)

// broadcaster broadcasts signals to registered signal listeners.
// All methods on the broadcaster are concurrency-safe.
type broadcaster struct {
	// This lock is used to protect all fields of broadcaster.
	// Methods on broadcaster should protect all concurrent access
	// by taking this lock when accessing its fields.
	// Conversely, this lock should NOT be taken outside of broadcaster.
	m sync.Mutex

	// last will contain a pointer to the last ShutdownSignal received, or
	// nil if none, if a new channel is created by Wait or Done, this last
	// signal will be immediately written to, this allows Wait or Done state
	// to be read after application stop
	last *ShutdownSignal

	// contains channels created by Done
	done []chan os.Signal

	// contains channels created by Wait
	wait []chan ShutdownSignal
}

func (b *broadcaster) reset() {
	b.m.Lock()
	defer b.m.Unlock()
	b.last = nil
}

// Done creates a new channel that will receive signals being broadcast
// via the broadcaster.
//
// If a signal has been received prior to the call of Done,
// the signal will be sent to the new channel.
func (b *broadcaster) Done() <-chan os.Signal {
	b.m.Lock()
	defer b.m.Unlock()

	ch := make(chan os.Signal, 1)
	// If we had received a signal prior to the call of done, send it's
	// os.Signal to the new channel.
	// However we still want to have the operating system notify signals to this
	// channel should the application receive another.
	if b.last != nil {
		ch <- b.last.Signal
	}
	b.done = append(b.done, ch)
	return ch
}

// Wait creates a new channel that will receive signals being broadcast
// via the broadcaster.
//
// If a signal has been received prior to the call of Wait,
// the signal will be sent to the new channel.
func (b *broadcaster) Wait() <-chan ShutdownSignal {
	b.m.Lock()
	defer b.m.Unlock()

	ch := make(chan ShutdownSignal, 1)

	if b.last != nil {
		ch <- *b.last
	}

	b.wait = append(b.wait, ch)
	return ch
}

// Broadcast sends the given signal to all channels that have been created
// via Done or Wait. It does not block on sending, and returns an unsentSignalError
// if any send did not go through.
func (b *broadcaster) Broadcast(signal ShutdownSignal) error {
	b.m.Lock()
	defer b.m.Unlock()

	b.last = &signal

	channels, unsent := b.broadcast(
		signal,
		b.broadcastDone,
		b.broadcastWait,
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

func (b *broadcaster) broadcast(
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

func (b *broadcaster) broadcastDone(signal ShutdownSignal) (int, int) {
	var unsent int

	for _, reader := range b.done {
		select {
		case reader <- signal.Signal:
		default:
			unsent++
		}
	}

	return len(b.done), unsent
}

func (b *broadcaster) broadcastWait(signal ShutdownSignal) (int, int) {
	var unsent int

	for _, reader := range b.wait {
		select {
		case reader <- signal:
		default:
			unsent++
		}
	}

	return len(b.wait), unsent
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
