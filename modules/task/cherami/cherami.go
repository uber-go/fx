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

package cherami

import (
	"context"
	"sync"
	"time"

	"go.uber.org/fx/config"
	"go.uber.org/fx/modules/task"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
	"github.com/uber-go/tally"
	"github.com/uber/cherami-client-go/client/cherami"
	cherami_gen "github.com/uber/cherami-thrift/.generated/go/cherami"
	"go.uber.org/zap"
)

type state int32

const (
	_initialized state = iota
	_running
	_stopped
	_pathPrefix    = "/uberfx_async/"
	_ctxKey        = "ctxKey"
	_operationName = "task.Run"
)

var (
	_hyperbahnMu                          sync.RWMutex
	_hyperbahnHostsFile                   string
	_cheramiClientFunc                          = cherami.NewHyperbahnClient
	_consumedMessagesRetentionInSeconds   int32 = 1 * 24 * 60 * 60 // 1 day
	_unconsumedMessagesRetentionInSeconds int32 = 7 * 24 * 60 * 60 // 7 days
	_defaultClientConfig                        = clientConfig{
		ConsumerName:       "uberfx-async",
		PrefetchCount:      10,
		ConsumeWorkerCount: 10,
		Timeout:            time.Second,
		DeploymentCluster:  "staging",
		CgTimeoutInSeconds: 60,
	}
)

type clientConfig struct {
	Destination        string
	ConsumerGroup      string
	ConsumerName       string
	PrefetchCount      int
	ConsumeWorkerCount int
	Timeout            time.Duration
	DeploymentCluster  string
	CgTimeoutInSeconds int32
}

// Backend holds cherami data
type Backend struct {
	client      cherami.Client
	publisher   cherami.Publisher
	consumer    cherami.Consumer
	deliveryCh  chan cherami.Delivery
	config      clientConfig
	logger      *zap.Logger
	scope       tally.Scope
	state       state
	stateMu     sync.RWMutex
	taskSuccess tally.Counter
	taskFailure tally.Counter
	tracer      opentracing.Tracer
	ctxEncoder  task.ContextEncoding
}

// RegisterHyperbahnBootstrapFile registers the hyperbahn bootstrap filename required for cherami
func RegisterHyperbahnBootstrapFile(filename string) {
	_hyperbahnMu.Lock()
	defer _hyperbahnMu.Unlock()
	_hyperbahnHostsFile = filename
}

// NewBackend creates a Cherami client backend
func NewBackend(
	cfg config.Provider,
	metrics tally.Scope,
	tracer opentracing.Tracer,
) (task.Backend, error) {
	serviceName := cfg.Get(config.ServiceNameKey).AsString()
	cc, err := createClientConfig(serviceName, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse config for cherami")
	}
	var ownerEmail string
	if err = cfg.Get(config.ServiceOwnerKey).Populate(&ownerEmail); err != nil {
		return nil, errors.Wrap(err, "unable to parse owner for cherami")
	}
	return newBackendWithConfig(serviceName, metrics, tracer, cc, ownerEmail)
}

func createClientConfig(serviceName string, cfg config.Provider) (clientConfig, error) {
	config := _defaultClientConfig
	// Set default destination and consumer group with the service name
	config.Destination = _pathPrefix + serviceName
	config.ConsumerGroup = _pathPrefix + serviceName + "_cg"
	// Preference to keys specified in config, they will be over-written
	// TODO: Might change based on module naming decision from fx public
	err := cfg.Get("modules").Get("task").Get("cherami").Populate(&config)
	return config, err
}

// newBackendWithConfig creates a Cherami client backend with specified config
func newBackendWithConfig(
	serviceName string,
	metrics tally.Scope,
	tracer opentracing.Tracer,
	cc clientConfig,
	ownerEmail string,
) (task.Backend, error) {
	// Create Cherami client TODO: Configure with reporter
	_hyperbahnMu.RLock()
	client, err := _cheramiClientFunc(
		serviceName, _hyperbahnHostsFile,
		&cherami.ClientOptions{DeploymentStr: cc.DeploymentCluster, Timeout: cc.Timeout},
	)
	defer _hyperbahnMu.RUnlock()
	if err != nil {
		return nil, errors.Wrapf(
			err, "unable to initialize cherami client for service: %q", serviceName,
		)
	}

	if err = createDestination(client, cc, ownerEmail); err != nil {
		return nil, err
	}
	if err = createConsumerGroup(client, cc, ownerEmail); err != nil {
		return nil, err
	}

	// Create message publisher via client
	publisher := client.CreatePublisher(&cherami.CreatePublisherRequest{
		Path: cc.Destination,
	})

	// Create message consumer via client
	consumer := client.CreateConsumer(&cherami.CreateConsumerRequest{
		Path:              cc.Destination,
		ConsumerGroupName: cc.ConsumerGroup,
		ConsumerName:      cc.ConsumerName,
		PrefetchCount:     cc.PrefetchCount,
		Options: &cherami.ClientOptions{
			Timeout: cc.Timeout,
		},
	})
	deliveryCh := make(chan cherami.Delivery, cc.PrefetchCount)
	scope := metrics.SubScope("cherami")

	return &Backend{
		client:      client,
		publisher:   publisher,
		consumer:    consumer,
		deliveryCh:  deliveryCh,
		config:      cc,
		logger:      ulog.Logger(context.Background()),
		scope:       scope,
		taskSuccess: scope.Counter("task.success"),
		taskFailure: scope.Counter("task.fail"),
		tracer:      tracer,
		ctxEncoder:  task.ContextEncoding{Tracer: tracer},
	}, nil
}

func createDestination(client cherami.Client, cc clientConfig, ownerEmail string) error {
	if _, err := client.CreateDestination(
		&cherami_gen.CreateDestinationRequest{
			Path: &cc.Destination,
			ConsumedMessagesRetention:   &_consumedMessagesRetentionInSeconds,
			UnconsumedMessagesRetention: &_unconsumedMessagesRetentionInSeconds,
			OwnerEmail:                  &ownerEmail,
		},
	); err != nil && !alreadyExistsError(err) {
		return errors.Wrapf(err, "unable to create destination: %q", cc.Destination)
	}
	return nil
}

func createConsumerGroup(client cherami.Client, cc clientConfig, ownerEmail string) error {
	if _, err := client.CreateConsumerGroup(
		&cherami_gen.CreateConsumerGroupRequest{
			DestinationPath:      &cc.Destination,
			ConsumerGroupName:    &cc.ConsumerGroup,
			OwnerEmail:           &ownerEmail,
			LockTimeoutInSeconds: &cc.CgTimeoutInSeconds,
		},
	); err != nil && !alreadyExistsError(err) {
		return errors.Wrapf(err, "unable to create consumer group: %q", cc.ConsumerGroup)
	}
	return nil
}

func alreadyExistsError(err error) bool {
	_, ok := err.(*cherami_gen.EntityAlreadyExistsError)
	return ok
}

func (b *Backend) setState(state state) {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()
	b.state = state
}

// Start the cherami pubsub
func (b *Backend) Start() error {
	var err error
	b.stateMu.RLock()
	if b.state == _running {
		b.stateMu.RUnlock()
		return errors.New("cannot start when module is already running")
	} else if b.state == _stopped {
		b.stateMu.RUnlock()
		return errors.New("cannot start when module has been stopped")
	}
	b.stateMu.RUnlock()
	if err := b.publisher.Open(); err != nil {
		return err
	}
	b.deliveryCh, err = b.consumer.Open(b.deliveryCh)
	if err != nil {
		return err
	}
	b.stateMu.Lock()
	b.state = _running
	b.stateMu.Unlock()
	return nil
}

// ExecuteAsync spins off workers to consume messages and execute tasks
func (b *Backend) ExecuteAsync() error {
	for i := 0; i < b.config.ConsumeWorkerCount; i++ {
		go b.consumeAndExecute()
	}
	return nil
}

func (b *Backend) consumeAndExecute() {
	defer func() {
		if r := recover(); r != nil {
			b.taskFailure.Inc(1)
			ulog.Logger(context.Background()).Error(
				"ExecuteAsync recovered from panic",
				zap.Any("msg", r),
			)
			b.consumeAndExecute()
		}
	}()

	for delivery := range b.deliveryCh {
		messageData := delivery.GetMessage().GetPayload().GetData()
		b.withContext(delivery, func(ctx context.Context) {
			// TODO (madhu): Only specific errors should be retried
			if err := task.Run(ctx, messageData); err != nil {
				ulog.Logger(ctx).Error("Task run failed", zap.Error(err))
				b.taskFailure.Inc(1)
				if err := delivery.Nack(); err != nil {
					ulog.Logger(ctx).Error("Delivery Nack failed", zap.Error(err))
				}
			} else {
				b.taskSuccess.Inc(1)
				if err = delivery.Ack(); err != nil {
					ulog.Logger(ctx).Error("Task ack to cherami failed", zap.Error(err))
				}
			}
		})
	}
}

func (b *Backend) withContext(delivery cherami.Delivery, f func(context.Context)) {
	ctxData := delivery.GetMessage().GetPayload().GetUserContext()
	ctx := context.Background()
	if ctxVal, ok := ctxData[_ctxKey]; ok {
		if spanCtx, err := b.ctxEncoder.Unmarshal([]byte(ctxVal)); err != nil {
			ulog.Logger(ctx).Error("Unable to decode context", zap.Error(err))
			b.taskFailure.Inc(1)
			if err := delivery.Nack(); err != nil {
				ulog.Logger(ctx).Error("Delivery Nack failed", zap.Error(err))
			}
		} else {
			var span opentracing.Span
			span = b.tracer.StartSpan(_operationName, ext.RPCServerOption(spanCtx))
			defer span.Finish()
			f(opentracing.ContextWithSpan(ctx, span))
		}

		return
	}

	f(ctx)
}

// IsRunning returns true if backend is running
func (b *Backend) isRunning() bool {
	b.stateMu.RLock()
	defer b.stateMu.RUnlock()
	return b.state == _running
}

// Enqueue sends the message to cherami
func (b *Backend) Enqueue(ctx context.Context, message []byte) error {
	ctxBytes, err := b.ctxEncoder.Marshal(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to encode context")
	}

	ctxMap := make(map[string]string)
	if len(ctxBytes) > 0 {
		ctxMap[_ctxKey] = string(ctxBytes)
	}

	receipt := b.publisher.Publish(&cherami.PublisherMessage{
		Data:        message,
		UserContext: ctxMap,
	})

	return receipt.Error
}

// Encoder returns an encoder for cherami messages
func (b *Backend) Encoder() task.Encoding {
	return task.GobEncoding{}
}

// Stop closes and cleans up cherami resources
func (b *Backend) Stop() error {
	b.stateMu.Lock()
	b.state = _stopped
	b.stateMu.Unlock()
	b.publisher.Close()
	b.consumer.Close()
	if b.deliveryCh != nil {
		close(b.deliveryCh)
	}
	b.client.Close()
	return nil
}
