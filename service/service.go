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

package service

// A State represents the state of a service
type State int

const (
	// Uninitialized means a service has not yet been initialized
	Uninitialized = State(iota)
	// Initialized means a service has been initialized
	Initialized
	// Starting represents a service in the process of starting
	Starting
	// Running represents a running service
	Running
	// Stopping represents a service in the process of stopping
	Stopping
	// Stopped represents a service that has been shut down
	Stopped
)

// A Manager encapsulates service ownership
type Manager interface {
	// Start service is used for blocking the call on service start. Start will block the
	// call and yield the control to the service lifecyce manager. No code will be executed
	//after call to Start() the service.
	Start()

	// StartAsync service is used as a non-blocking the call on service start. StartAsync will
	// return the call to the caller with a Control to listen on channels
	// and trigger manual shutdown.
	StartAsync() Control
	Stop(reason string, exitCode int) error
}

// Control keeps the listening channels from the service startup
type Control struct {
	ExitChan     chan Exit
	ReadyChan    chan struct{}
	ServiceError error
}

// Exit is a signal for a service that needs to exit
type Exit struct {
	Reason   string
	Error    error
	ExitCode int
}

// AfterStart will create an observer that will execute f() immediately after service starts.
func AfterStart(f func()) Observer {
	return niladicStart(f)
}

type niladicStart func()

func (n niladicStart) OnInit() error                  { return nil }
func (n niladicStart) OnShutdown(reason Exit)         {}
func (n niladicStart) OnCriticalError(err error) bool { return true }
func (n niladicStart) OnStateChange(old State, curr State) {
	if old == Starting && curr == Running {
		n()
	}
}
