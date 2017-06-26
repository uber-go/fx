package fxlog

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type spy struct {
	*bytes.Buffer
}

func newSpy() *spy {
	return &spy{bytes.NewBuffer(nil)}
}

func (s *spy) Printf(format string, is ...interface{}) {
	fmt.Fprintln(s, fmt.Sprintf(format, is...))
}

// stubs the exit call, returns a function that restores a real exit function
// and asserts that the stub was called.
func stubExit() func(testing.TB) {
	prev := _exit
	var called bool
	_exit = func() { called = true }
	return func(t testing.TB) {
		assert.True(t, called, "Exit wasn't called.")
		_exit = prev
	}
}

func TestNew(t *testing.T) {
	assert.NotPanics(t, func() { New() })
}

func TestPrint(t *testing.T) {
	sink := newSpy()
	logger := &Logger{sink}

	t.Run("println", func(t *testing.T) {
		sink.Reset()
		logger.Println("foo")
		assert.Equal(t, "[Fx] foo\n", sink.String())
	})

	t.Run("printf", func(t *testing.T) {
		sink.Reset()
		logger.Printf("foo %d", 42)
		assert.Equal(t, "[Fx] foo 42\n", sink.String())
	})

	t.Run("printProvide", func(t *testing.T) {
		sink.Reset()
		logger.PrintProvide(bytes.NewBuffer)
		assert.Equal(t, "[Fx] PROVIDE\t*bytes.Buffer <= bytes.NewBuffer()\n", sink.String())
	})

	t.Run("printProvideInvalid", func(t *testing.T) {
		sink.Reset()
		// No logging on invalid provides, since we're already logging an error
		// elsewhere.
		logger.PrintProvide(bytes.NewBuffer(nil))
		assert.Equal(t, "", sink.String())
	})

	t.Run("printSignal", func(t *testing.T) {
		sink.Reset()
		logger.PrintSignal(os.Interrupt)
		assert.Equal(t, "[Fx] INTERRUPT\n", sink.String())
	})
}

func TestPanic(t *testing.T) {
	sink := newSpy()
	logger := &Logger{sink}
	assert.Panics(t, func() { logger.Panic(errors.New("foo")) })
	assert.Equal(t, "[Fx] foo\n", sink.String())
}

func TestFatal(t *testing.T) {
	sink := newSpy()
	logger := &Logger{sink}

	undo := stubExit()
	defer undo(t)

	logger.Fatalf("foo %d", 42)
	assert.Equal(t, "[Fx] foo 42\n", sink.String())
}
