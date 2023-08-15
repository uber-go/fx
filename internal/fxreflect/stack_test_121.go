//go:build go1.21
// +build go1.21

package fxreflect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepStack(t *testing.T) {
	// Introduce a few frames.
	frames := func() []Frame {
		return func() []Frame {
			return CallerStack(0, 0)
		}()
	}()

	require.True(t, len(frames) > 3, "expected at least three frames")
	for i, name := range []string{"func2.TestStack.func2.1.func2", "func2.1", "func2"} {
		f := frames[i]
		assert.Equal(t, "go.uber.org/fx/internal/fxreflect.TestStack."+name, f.Function)
		assert.Contains(t, f.File, "internal/fxreflect/stack_test.go")
		assert.NotZero(t, f.Line)
	}
}
