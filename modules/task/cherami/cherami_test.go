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
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/fx/auth"
	"go.uber.org/fx/config"
	cherami_mocks "go.uber.org/fx/mocks/modules/task/cherami"
	"go.uber.org/fx/modules/task"
	"go.uber.org/fx/service"
	"go.uber.org/fx/testutils"
	"go.uber.org/fx/ulog"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uber/cherami-client-go/client/cherami"
	cherami_gen "github.com/uber/cherami-thrift/.generated/go/cherami"
)

var (
	_host = service.NopHostConfigured(
		auth.NopClient, ulog.Logger(context.Background()), opentracing.NoopTracer{},
	)
	_pathName = _pathPrefix + _host.Name()
	_cgName   = _pathPrefix + _host.Name() + "_cg"
)

func TestBackendWorkflow(t *testing.T) {
	m := newMock()
	defer m.AssertExpectations(t)
	zapLogger, buf := testutils.GetLockedInMemoryLogger()
	defer ulog.SetLogger(zapLogger)()
	bknd := createNewBackend(t, m)
	assert.NotNil(t, bknd.Encoder())
	deliveryCh, err := startBackend(t, m, bknd, nil, nil)
	require.NoError(t, err)
	assert.True(t, bknd.(*Backend).isRunning())
	require.NoError(t, bknd.ExecuteAsync())

	publish(t, m, bknd, deliveryCh)
	publish(t, m, bknd, deliveryCh)
	time.Sleep(10 * time.Millisecond)
	stopBackend(t, m, bknd)
	lines := buf.Lines()
	require.Equal(t, 2, len(lines))
	for _, line := range lines {
		assert.Contains(t, line, "forget to register")
	}
}

func TestBackendWorkflowWorkerPanic(t *testing.T) {
	m := newMock()
	defer m.AssertExpectations(t)
	zapLogger, buf := testutils.GetLockedInMemoryLogger()
	defer ulog.SetLogger(zapLogger)()
	bknd := createNewBackend(t, m)
	deliveryCh, err := startBackend(t, m, bknd, nil, nil)
	require.NoError(t, err)
	assert.True(t, bknd.(*Backend).isRunning())
	require.NoError(t, bknd.ExecuteAsync())
	// Panic on ConsumeWorkerCount and make sure workers are still alive to consume messages
	for i := 0; i < _defaultClientConfig.ConsumeWorkerCount; i++ {
		deliveryCh <- m.Delivery
		m.Delivery.On("GetMessage").Return(
			&cherami_gen.ConsumerMessage{
				Payload: &cherami_gen.PutMessage{Data: []byte("Hello")},
			},
		)
		m.Delivery.On("Nack").Run(func(mock.Arguments) { panic("nack panic") })
	}
	// Publish valid message
	publish(t, m, bknd, deliveryCh)
	time.Sleep(10 * time.Millisecond)
	assert.True(t, bknd.(*Backend).isRunning())
	stopBackend(t, m, bknd)
	// Nack panics are sent for a count of _numWorkers and 1 valid publish. Make sure they are
	// all processed
	lines := buf.Lines()
	var ct int
	for _, line := range lines {
		if strings.Contains(line, "forget to register") {
			ct++
		}
	}
	require.Equal(t, _defaultClientConfig.ConsumeWorkerCount+1, ct)
}

func TestBackendWorkflowStateLocks(t *testing.T) {
	m := newMock()
	defer m.AssertExpectations(t)
	bknd := createNewBackend(t, m)
	assert.NotNil(t, bknd.Encoder())
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		startBackend(t, m, bknd, nil, nil)
		for bknd.(*Backend).isRunning() {
			runtime.Gosched()
		}
		wg.Done()
	}()
	go func() {
		for !bknd.(*Backend).isRunning() {
			runtime.Gosched()
		}
		stopBackend(t, m, bknd)
		wg.Done()
	}()
	wg.Wait()
}

func TestNewBackendClientError(t *testing.T) {
	_cheramiClientFunc = func(
		serviceName string, bootstrapFile string, options *cherami.ClientOptions,
	) (cherami.Client, error) {
		return nil, errors.New("failure")
	}
	checkNewBackendError(t, "client for service: \"dummy\": failure")
}

func TestNewBackendReadEntityNotExistsCreateDestError(t *testing.T) {
	m := newMock()
	defer m.AssertExpectations(t)
	setupHappyClientFunc(m)
	setupDest(m, _pathName, errors.New("create error"))
	checkNewBackendError(t, "create destination: \"/uberfx_async/dummy\"")
}

func TestNewBackendEntityNotExistsCreateCgError(t *testing.T) {
	m := newMock()
	defer m.AssertExpectations(t)
	setupHappyClientFunc(m)
	setupDest(m, _pathName, nil)
	setupCg(m, _pathName, _cgName, errors.New("create error"))
	checkNewBackendError(t, "create consumer group: \"/uberfx_async/dummy_cg\"")
}

func TestNewBackendCreateEntityExistsSuccess(t *testing.T) {
	m := newMock()
	defer m.AssertExpectations(t)
	setupHappyClientFunc(m)
	setupDest(m, _pathName, &cherami_gen.EntityAlreadyExistsError{})
	setupCg(m, _pathName, _cgName, &cherami_gen.EntityAlreadyExistsError{})
	setupPublisherConsumer(m, _pathName, _cgName)
	bknd, err := NewBackend(_host)
	require.NoError(t, err)
	assert.NotNil(t, bknd)
	assert.False(t, bknd.(*Backend).isRunning())
}

func TestNewBackendCreateWithConfiguredHost(t *testing.T) {
	data := []byte(`
name: dummy
owner: owner@owner.com
modules:
  task:
    cherami:
      destination: /my_dest/
      consumerGroup: /my_dest_cg/
      deploymentCluster: dev
      cgTimeoutInSeconds: 15
`)
	path := "/my_dest/"
	cg := "/my_dest_cg/"
	provider := config.NewYAMLProviderFromBytes(data)
	host := service.NopHostWithConfig(provider)
	RegisterHyperbahnBootstrapFile("hyperbahn-filename")

	// Setup mock and function calls
	m := newMock()
	defer m.AssertExpectations(t)
	_cheramiClientFunc = func(
		serviceName string, bootstrapFile string, options *cherami.ClientOptions,
	) (cherami.Client, error) {
		require.Equal(t, "dev", options.DeploymentStr)
		require.Equal(t, "hyperbahn-filename", bootstrapFile)
		return m.Client, nil
	}
	m.Client.On(
		"CreateDestination", mock.MatchedBy(func(request *cherami_gen.CreateDestinationRequest) bool {
			return request.GetPath() == path &&
				request.GetOwnerEmail() == "owner@owner.com" &&
				request.GetConsumedMessagesRetention() == 86400 &&
				request.GetUnconsumedMessagesRetention() == 604800
		}),
	).Return(nil, nil)
	m.Client.On(
		"CreateConsumerGroup", mock.MatchedBy(
			func(request *cherami_gen.CreateConsumerGroupRequest) bool {
				return request.GetDestinationPath() == path &&
					request.GetConsumerGroupName() == cg &&
					request.GetOwnerEmail() == "owner@owner.com" &&
					request.GetLockTimeoutInSeconds() == 15
			}),
	).Return(nil, nil)
	setupPublisherConsumer(m, path, cg)

	// Create backend
	bknd, err := NewBackend(host)
	require.NoError(t, err)
	assert.NotNil(t, bknd)
	assert.False(t, bknd.(*Backend).isRunning())
}

func checkNewBackendError(t *testing.T, errStr string) {
	bknd, err := NewBackend(_host)
	require.Error(t, err)
	assert.Contains(t, err.Error(), errStr)
	assert.Nil(t, bknd)
}

func TestStartBackendInvalidStateError(t *testing.T) {
	stateToError := map[state]string{_running: "already running", _stopped: "has been stopped"}
	for state, errStr := range stateToError {
		m := newMock()
		bknd := createNewBackend(t, m)
		bknd.(*Backend).setState(state)
		err := bknd.Start()
		assert.Contains(t, err.Error(), errStr)
	}
}

func TestStartBackendOpenPublisherError(t *testing.T) {
	m := newMock()
	defer m.AssertExpectations(t)
	bknd := createNewBackend(t, m)
	errStr := "publish error"
	_, err := startBackend(t, m, bknd, errors.New(errStr), nil)
	assert.False(t, bknd.(*Backend).isRunning())
	assert.Contains(t, err.Error(), errStr)
}

func TestStartBackendOpenConsumerError(t *testing.T) {
	m := newMock()
	defer m.AssertExpectations(t)
	bknd := createNewBackend(t, m)
	errStr := "consume error"
	_, err := startBackend(t, m, bknd, nil, errors.New(errStr))
	assert.False(t, bknd.(*Backend).isRunning())
	assert.Contains(t, err.Error(), errStr)
}

type cheramiMock struct {
	Client   *cherami_mocks.Client
	Pub      *cherami_mocks.Publisher
	Con      *cherami_mocks.Consumer
	Delivery *cherami_mocks.Delivery
}

func newMock() *cheramiMock {
	return &cheramiMock{
		Client:   &cherami_mocks.Client{},
		Pub:      &cherami_mocks.Publisher{},
		Con:      &cherami_mocks.Consumer{},
		Delivery: &cherami_mocks.Delivery{},
	}
}

func (m *cheramiMock) AssertExpectations(t *testing.T) {
	m.Client.AssertExpectations(t)
	m.Pub.AssertExpectations(t)
	m.Con.AssertExpectations(t)
	m.Delivery.AssertExpectations(t)
}

func createNewBackend(t *testing.T, m *cheramiMock) task.Backend {
	setupHappyClientFunc(m)
	setupDest(m, _pathName, nil)
	setupCg(m, _pathName, _cgName, nil)
	setupPublisherConsumer(m, _pathName, _cgName)
	bknd, err := NewBackend(_host)
	require.NoError(t, err)
	assert.NotNil(t, bknd)
	assert.False(t, bknd.(*Backend).isRunning())
	return bknd
}

func stopBackend(t *testing.T, m *cheramiMock, bknd task.Backend) {
	m.Pub.On("Close")
	m.Con.On("Close")
	m.Client.On("Close")
	require.NoError(t, bknd.Stop())
}

func setupHappyClientFunc(m *cheramiMock) {
	_cheramiClientFunc = func(
		serviceName string, bootstrapFile string, options *cherami.ClientOptions,
	) (cherami.Client, error) {
		return m.Client, nil
	}
}

func setupDest(m *cheramiMock, pathName string, createErr error) {
	m.Client.On(
		"CreateDestination", mock.MatchedBy(func(request *cherami_gen.CreateDestinationRequest) bool {
			return request.GetPath() == pathName &&
				request.GetConsumedMessagesRetention() == 86400 &&
				request.GetUnconsumedMessagesRetention() == 604800
		}),
	).Return(nil, createErr)
}

func setupCg(m *cheramiMock, pathName string, cgName string, createErr error) {
	m.Client.On(
		"CreateConsumerGroup", mock.MatchedBy(
			func(request *cherami_gen.CreateConsumerGroupRequest) bool {
				return request.GetDestinationPath() == pathName &&
					request.GetConsumerGroupName() == cgName &&
					request.GetLockTimeoutInSeconds() == 60
			}),
	).Return(nil, createErr)
}

func setupPublisherConsumer(m *cheramiMock, pathName, cgName string) {
	m.Client.On("CreatePublisher", &cherami.CreatePublisherRequest{Path: pathName}).Return(m.Pub)
	m.Client.On("CreateConsumer", &cherami.CreateConsumerRequest{
		Path:              pathName,
		ConsumerGroupName: cgName,
		ConsumerName:      _defaultClientConfig.ConsumerName,
		PrefetchCount:     _defaultClientConfig.PrefetchCount,
		Options: &cherami.ClientOptions{
			Timeout: _defaultClientConfig.Timeout,
		},
	}).Return(m.Con)
}

func startBackend(
	t *testing.T, m *cheramiMock, bknd task.Backend, publishError error, consumeError error,
) (chan cherami.Delivery, error) {
	var deliveryCh chan cherami.Delivery
	m.Pub.On("Open").Return(publishError)
	if publishError == nil {
		deliveryCh = make(chan cherami.Delivery, _defaultClientConfig.PrefetchCount)
		m.Con.On("Open", bknd.(*Backend).deliveryCh).Return(deliveryCh, consumeError)
	}
	err := bknd.Start()
	return deliveryCh, err
}

func publish(t *testing.T, m *cheramiMock, bknd task.Backend, deliveryCh chan cherami.Delivery) {
	msg := []byte("Hello")
	m.Pub.On(
		"Publish", &cherami.PublisherMessage{Data: msg, UserContext: map[string]string{}},
	).Run(
		func(mock.Arguments) { deliveryCh <- m.Delivery },
	).Return(&cherami.PublisherReceipt{})
	m.Delivery.On("GetMessage").Return(
		&cherami_gen.ConsumerMessage{
			Payload: &cherami_gen.PutMessage{Data: msg},
		},
	)
	m.Delivery.On("Nack").Return(nil)
	require.NoError(t, bknd.Enqueue(context.Background(), msg))
}
