package fxreflect

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReturnTypes(t *testing.T) {
	t.Run("Primitive", func(t *testing.T) {
		fn := func() (int, string) {
			return 0, ""
		}
		assert.Equal(t, []string{"int", "string"}, ReturnTypes(fn))
	})
	t.Run("Pointer", func(t *testing.T) {
		type s struct{}
		fn := func() *s {
			return &s{}
		}
		assert.Equal(t, []string{"*fxreflect.s"}, ReturnTypes(fn))
	})
	t.Run("Interface", func(t *testing.T) {
		fn := func() hollerer {
			return impl{}
		}
		assert.Equal(t, []string{"fxreflect.hollerer"}, ReturnTypes(fn))
	})
	t.Run("SkipsErr", func(t *testing.T) {
		fn := func() (string, error) {
			return "", errors.New("err")
		}
		assert.Equal(t, []string{"string"}, ReturnTypes(fn))
	})
}

type hollerer interface {
	Holler()
}
type impl struct{}

func (impl) Holler() {}

func TestCaller(t *testing.T) {
	t.Run("CalledByTestingRunner", func(t *testing.T) {
		assert.Equal(t, "testing.tRunner", Caller())
	})
}

func someFunc() {}

func TestFuncName(t *testing.T) {
	assert.Equal(t, "go.uber.org/fx/internal/fxreflect.someFunc()", FuncName(someFunc))
}
