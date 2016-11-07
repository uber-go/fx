// Copyright (c) 2016 Uber Technologies, Inc.
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

package ulog

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"go.uber.org/fx/core/config"

	"github.com/uber-go/zap"
)

// Configuration for logging with uberfx
type Configuration struct {
	Level   string
	File    *FileConfiguration
	Stdout  bool
	Verbose bool

	prefixWithFileLine *bool `yaml:"prefix_with_fileline"`
	TextFormatter      *bool // use TextFormatter (default json)
}

// FileConfiguration describes the properties needed to log to a file
type FileConfiguration struct {
	Enabled   bool
	Directory string
	FileName  string
}

type baselogger struct {
	log        zap.Logger
	initFields []interface{}
}

// Log is the Uberfx wrapper for underlying logging service
type Log interface {
	// Configure Log object with the provided log.Configuration
	Configure(Configuration)
	// Create a child logger with some context
	With(...interface{}) Log
	SetLevel(zap.Level)
	SetLogger(zap.Logger)
	Check(zap.Level, string) *zap.CheckedMessage
	RawLogger() zap.Logger

	Log(zap.Level, string, ...interface{})
	Debug(string, ...interface{})
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
	Panic(string, ...interface{})
	Fatal(string, ...interface{})
	DFatal(string, ...interface{})
}

var development = strings.Contains(config.GetEnvironment(), "development")

var std = defaultLogger()

func defaultLogger() Log {
	return &baselogger{
		log:        zap.New(zap.NewJSONEncoder()),
		initFields: nil,
	}
}

// Logger returns the package-level logger configured for the service
func Logger(fields ...interface{}) Log {
	return &baselogger{
		log:        std.(*baselogger).log,
		initFields: fields,
	}
}

// Configure the package-level logger based on provided configuration
func Configure(cfg Configuration) {
	std.Configure(cfg)
}

// Configure the calling logger based on provided configuration
func (l *baselogger) Configure(cfg Configuration) {
	options := make([]zap.Option, 0, 3)
	if cfg.Verbose {
		options = append(options, zap.DebugLevel)
	} else {
		l.With(zap.String("Level", cfg.Level)).Debug("setting log level")
		var lv zap.Level
		err := lv.UnmarshalText([]byte(cfg.Level))
		if err != nil {
			l.With(zap.String("Level", cfg.Level)).Debug("cannot parse log level. setting to Debug as default")
			options = append(options, zap.DebugLevel)
		} else {
			options = append(options, lv)
		}
	}
	if cfg.File == nil || !cfg.File.Enabled {
		options = append(options, zap.Output(zap.AddSync(os.Stdout)))
	} else {
		options = append(options, zap.Output(l.fileOutput(cfg.File, cfg.Stdout, cfg.Verbose)))
	}

	if cfg.TextFormatter != nil && *cfg.TextFormatter || cfg.TextFormatter == nil && development {
		l.SetLogger(zap.New(zap.NewTextEncoder(), options...))
		return
	}
	l.SetLogger(zap.New(zap.NewJSONEncoder(), options...))
}

func (l *baselogger) fileOutput(cfg *FileConfiguration, stdout bool, verbose bool) zap.WriteSyncer {
	fileLoc := path.Join(cfg.Directory, cfg.FileName)
	l.Debug("adding log file output to zap")
	err := os.MkdirAll(cfg.Directory, os.FileMode(0755))
	if err != nil {
		l.Fatal("failed to create log directory: ", err)
	}
	file, err := os.OpenFile(fileLoc, os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.FileMode(0755))
	if err != nil {
		l.With(zap.Error(err)).Fatal("failed to open log file for writing.")
	}
	l.With(zap.String("filename", fileLoc)).Debug("logfile created successfully")
	if verbose || stdout {
		return zap.AddSync(io.MultiWriter(os.Stdout, file))
	}
	return zap.AddSync(file)
}

// SetLogger allows users to inject underlying zap.Logger. Use this
// for test, or or if you fancy your own logging setup
func (l *baselogger) SetLogger(log zap.Logger) {
	l.log = log
}

// RawLogger returns underneath zap implementation for use
func (l *baselogger) RawLogger() zap.Logger {
	return l.log
}

func (l *baselogger) SetLevel(level zap.Level) {
	l.log.SetLevel(level)
}

func (l *baselogger) With(fields ...interface{}) Log {
	return &baselogger{
		log:        l.log.With(l.compileLogFields(fields...)...),
		initFields: l.initFields,
	}
}

func (l *baselogger) Check(level zap.Level, expr string) *zap.CheckedMessage {
	return l.log.Check(level, expr)
}

func (l *baselogger) Debug(msg string, args ...interface{}) {
	l.Log(zap.DebugLevel, msg, args...)
}

func (l *baselogger) Info(msg string, args ...interface{}) {
	l.Log(zap.InfoLevel, msg, args...)
}

func (l *baselogger) Warn(msg string, args ...interface{}) {
	l.Log(zap.WarnLevel, msg, args...)
}

func (l *baselogger) Error(msg string, args ...interface{}) {
	l.Log(zap.ErrorLevel, msg, args...)
}

func (l *baselogger) Panic(msg string, args ...interface{}) {
	l.Log(zap.PanicLevel, msg, args...)
}

func (l *baselogger) Fatal(msg string, args ...interface{}) {
	l.Log(zap.FatalLevel, msg, args...)
}

func (l *baselogger) DFatal(msg string, args ...interface{}) {
	l.log.DFatal(msg, l.compileLogFields(args)...)
}

func (l *baselogger) Log(lvl zap.Level, msg string, args ...interface{}) {
	if cm := l.Check(lvl, msg); cm.OK() {
		cm.Write(l.compileLogFields(args...)...)
	}
}

func (l *baselogger) compileLogFields(args ...interface{}) []zap.Field {
	var fields []interface{}
	fields = append(fields, l.initFields...)
	fields = append(fields, args...)
	return l.fieldsConversion(fields...)
}

func (l *baselogger) fieldsConversion(args ...interface{}) []zap.Field {
	fields := make([]zap.Field, 0, len(args)/2)
	if len(args)%2 != 0 {
		fields = append(fields, zap.Error(fmt.Errorf("invalid number of arguments")))
		return fields
	}
	for idx := 0; idx < len(args); idx += 2 {
		if key, ok := args[idx].(string); ok {
			key = args[idx].(string)
			switch value := args[idx+1].(type) {
			case bool:
				fields = append(fields, zap.Bool(key, value))
			case float64:
				fields = append(fields, zap.Float64(key, value))
			case int:
				fields = append(fields, zap.Int(key, value))
			case int64:
				fields = append(fields, zap.Int64(key, value))
			case uint:
				fields = append(fields, zap.Uint(key, value))
			case uint64:
				fields = append(fields, zap.Uint64(key, value))
			case uintptr:
				fields = append(fields, zap.Uintptr(key, value))
			case string:
				fields = append(fields, zap.String(key, value))
			case time.Time:
				fields = append(fields, zap.Time(key, value))
			case time.Duration:
				fields = append(fields, zap.Duration(key, value))
			case zap.LogMarshaler:
				fields = append(fields, zap.Marshaler(key, value))
			case fmt.Stringer:
				fields = append(fields, zap.Stringer(key, value))
			case error:
				fields = append(fields, zap.Error(value))
			default:
				fields = append(fields, zap.Object(key, value))
			}
		}
	}
	return fields
}
