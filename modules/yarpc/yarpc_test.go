package yarpc

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"go.uber.org/fx/config"
	"go.uber.org/fx/metrics"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/dig"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	yhttp "go.uber.org/yarpc/transport/http"
	"go.uber.org/zap"
)

type testOnewayHandler struct {
	handler func(ctx context.Context, req *transport.Request) error
}

func (t *testOnewayHandler) HandleOneway(ctx context.Context, req *transport.Request) error {
	return t.handler(ctx, req)
}

func TestYARPC_HTTPTransportE2E(t *testing.T) {
	t.Parallel()

	di := dig.New()
	var dispatcher *yarpc.Dispatcher
	wg := sync.WaitGroup{}
	wg.Add(1)
	fn := func(d *yarpc.Dispatcher) (*Transports, error) {
		require.NotNil(t, d)
		dispatcher = d
		proc := []transport.Procedure{{
			Name:      "TestName",
			Service:   "ServiceName",
			Encoding:  "jsonShallNotBeUsed",
			Signature: "SaltAndPepper",
			HandlerSpec: transport.NewOnewayHandlerSpec(&testOnewayHandler{
				handler: func(ctx context.Context, req *transport.Request) error {
					assert.Equal(t, "SecretAgent", req.Caller)
					wg.Done()
					return nil
				},
			}),
		},
		}

		return &Transports{Ts: proc}, nil
	}

	module := New(fn)
	for _, component := range module.Constructor() {
		di.MustRegister(component)
	}

	cfg := map[string]interface{}{
		"name": "test",
		"modules.yarpc": map[string]interface{}{
			"inbounds": map[string]interface{}{
				"http": map[string]interface{}{
					"address": ":0",
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

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://%s", dispatcher.Introspect().Inbounds[0].Endpoint), &bytes.Buffer{})

	require.NoError(t, err)
	req.Header = http.Header{
		yhttp.ServiceHeader:   []string{"ServiceName"},
		yhttp.ProcedureHeader: []string{"TestName"},
		yhttp.CallerHeader:    []string{"SecretAgent"},
		yhttp.EncodingHeader:  []string{"jsonShallNotBeUsed"},
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	wg.Wait()
	require.NoError(t, module.Stop())
}
