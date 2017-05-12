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

// StubObserver may be used for tests or demonstration apps where you wish to
// observe state transitions without implementing your own observer
type StubObserver struct {
	state         State
	shutdown      bool
	init          bool
	criticalError bool
	initError     error
}

// OnInit is called when a service is initialized
func (s *StubObserver) OnInit() error {
	s.init = true
	return s.initError
}

// OnStateChange is called during state transitions
func (s *StubObserver) OnStateChange(old State, curr State) {
	s.state = curr
}

// OnShutdown is called for a shutdown
func (s *StubObserver) OnShutdown(reason Exit) {
	s.shutdown = true
}

// OnCriticalError is called whin the app is about to shut down
func (s *StubObserver) OnCriticalError(err error) bool {
	s.criticalError = true
	return false
}

// ObserverStub returns a stub instance of an Observer
func ObserverStub() Observer {
	return &StubObserver{}
}
