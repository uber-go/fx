package fx_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type TransientService struct {
	ID int64
}

func NewTransientService() *TransientService {
	return &TransientService{
		ID: rand.Int63(),
	}
}

type ConsumerA struct {
	Service *TransientService
}

type ConsumerB struct {
	Service *TransientService
}

func TestTransientProvider(t *testing.T) {
	t.Run("should create new instance for each resolution", func(t *testing.T) {
		app := fxtest.New(t,
			fx.Provide(
				fx.Transient(NewTransientService),
			),
			fx.Invoke(func(factory func() *TransientService) {
				i1 := factory()
				i2 := factory()
				assert.NotEqual(t, i1, i2)
				assert.NotEqual(t, i1.ID, i2.ID)
			}),
		)
		defer app.RequireStart().RequireStop()
	})

	t.Run("should create new instance when injected into different consumers", func(t *testing.T) {
		app := fxtest.New(t,
			fx.Provide(
				fx.Transient(NewTransientService),
				func(f func() *TransientService) *ConsumerA { return &ConsumerA{Service: f()} },
				func(f func() *TransientService) *ConsumerB { return &ConsumerB{Service: f()} },
			),
			fx.Invoke(func(a *ConsumerA, b *ConsumerB) {
				assert.NotEqual(t, a.Service, b.Service)
				assert.NotEqual(t, a.Service.ID, b.Service.ID)
			}),
		)
		defer app.RequireStart().RequireStop()
	})
}
