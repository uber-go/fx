package fxreflect

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsErr(t *testing.T) {
	t.Run("ReturnsTrue", func(t *testing.T) {
		assert.True(t, IsErr(reflect.TypeOf(errors.New("err"))))
	})
	t.Run("ReturnsFalse", func(t *testing.T) {
		type s struct{}
		assert.False(t, IsErr(reflect.TypeOf(&s{})))
	})
}

func TestReturnTypes(t *testing.T) {
	t.Run("Scalar", func(t *testing.T) {
		fn := func() (int, string) {
			return 0, ""
		}
		assert.Equal(t, []string{"int", "string"}, ReturnTypes(fn))
	})
	t.Run("Struct", func(t *testing.T) {
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
