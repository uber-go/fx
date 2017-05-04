package yarpc

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
	"go.uber.org/zap"
	"go.uber.org/yarpc"
)

func TestYARPC_HTTPTransportHealthAndStop(t *testing.T) {
	t.Parallel()

	di := dig.New()

	var dispatcher *yarpc.Dispatcher
	fn := func(d *yarpc.Dispatcher) (*Transports, error) {
		require.NotNil(t, d)
		dispatcher = d
		return &Transports{}, nil
	}

	module := New(fn)
	for _, component := range module.Constructor() {
		di.MustRegister(component)
	}

	cfg := map[string]interface{}{
		"name": "test",
		"modules.yarpc": map[string]interface{}{
			"inbounds": []interface{}{
				map[string]interface{}{
					"http": map[string]interface{}{
						"port": 0,
					},
				},
			},
		},
	}

	provider := config.NewStaticProvider(cfg)
	di.MustRegister(&provider)
	tracer := opentracing.Tracer(opentracing.NoopTracer{})
	di.MustRegister(&tracer)
	di.MustRegister(&metrics.NopScope)
	logger := zap.NewNop()
	di.MustRegister(logger)

	var starter *starter
	di.MustResolve(&starter)

	require.NotNil(t, dispatcher)
}