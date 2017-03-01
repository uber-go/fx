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

package ugrpc

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/fx/service"
	"go.uber.org/fx/ulog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	// LoggingUnaryServerInterceptor is a UnaryServerInterceptor that logs a request and response.
	LoggingUnaryServerInterceptor grpc.UnaryServerInterceptor = logUnaryRequest
)

func init() {
	grpclog.SetLogger(newLogger(zap.S()))
}

// Module returns a new ModuleCreateFunc for the given registration function.
func Module(registerFunc func(*grpc.Server), options ...grpc.ServerOption) service.ModuleCreateFunc {
	return func(host service.Host) (service.Module, error) {
		return newModule(host, registerFunc, options...)
	}
}

type config struct {
	Port uint16
}

type module struct {
	config *config
	server *grpc.Server
}

func newModule(host service.Host, registerFunc func(*grpc.Server), options ...grpc.ServerOption) (service.Module, error) {
	config := &config{}
	if err := host.Config().Scope("modules").Get(host.Name()).PopulateStruct(config); err != nil {
		return nil, err
	}
	server := grpc.NewServer(options...)
	registerFunc(server)
	return &module{config, server}, nil
}

func (m *module) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.Port))
	if err != nil {
		return err
	}
	go func() {
		if err := m.server.Serve(listener); err != nil {
			ulog.Logger(context.Background()).Error("grpc serve error", zap.Error(err))
		}
	}()
	return nil
}

func (m *module) Stop() error {
	m.server.Stop()
	return nil
}

type logger struct {
	*zap.SugaredLogger
}

func newLogger(sugaredLogger *zap.SugaredLogger) *logger {
	return &logger{sugaredLogger}
}

func (l *logger) Print(args ...interface{}) {
	l.Info(args...)
}

func (l *logger) Printf(format string, args ...interface{}) {
	l.Infof(format, args...)
}

func (l *logger) Println(args ...interface{}) {
	l.Info(args...)
}

func (l *logger) Fatalln(args ...interface{}) {
	l.Fatal(args...)
}

func logUnaryRequest(
	ctx context.Context,
	request interface{},
	unaryServerInfo *grpc.UnaryServerInfo,
	unaryHandler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	response, err := unaryHandler(ctx, request)
	duration := time.Since(start)
	fields := []zapcore.Field{
		zap.String("method", unaryServerInfo.FullMethod),
		zap.Any("request", request),
		zap.Duration("duration", duration),
	}
	if response != nil {
		fields = append(fields, zap.Any("response", response))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	ulog.Logger(ctx).Info("", fields...)
	return response, err
}
