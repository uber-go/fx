package http

import (
	"github.com/uber-go/uberfx/core"
	"github.com/uber-go/uberfx/core/config"
	"github.com/uber-go/uberfx/core/metrics"
	"github.com/uber-go/annotate"

	"encoding/json"
	"fmt"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	// goctx "golang.org/x/net/context"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"

	"net"
)

type HTTPContext struct {
	req *http.Request
	res http.ResponseWriter
}

func (hctx *HTTPContext) Request() *http.Request {
	return hctx.req
}

func (hctx *HTTPContext) Response() http.ResponseWriter {
	return hctx.res
}

// module

type HttpModule struct {
	core.ModuleBase
	title              string
	config             HttpConfig
	createHandlerFuncs []CreateHandlerFunc
	mux                sync.Mutex
	router             *mux.Router
	listener           net.Listener
}

var _ core.Module = &HttpModule{}

const HTTPModuleType = "HTTP"

type HttpConfig struct {
	core.ModuleConfig
	Port           int `yaml:"port"`
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

type CreateHandlerFunc func(structInfo *annotate.StructInfo) interface{}

func getConfigKey(name string) string {
	return fmt.Sprintf("modules.%s", name)
}

func Module(name string, moduleRoles []string) core.ModuleCreateFunc {
	return func(svc *core.Service) ([]core.Module, error) {
		if mod, err := newHttpModule(name, svc, moduleRoles); err != nil {
			return nil, err
		} else {
			return []core.Module{mod}, nil
		}
	}
}

func newHttpModule(name string, service *core.Service, roles []string) (*HttpModule, error) {
	// setup config defaults
	//
	cfg := &HttpConfig{
		Port:           3001,
		TimeoutSeconds: 60,
	}

	config.Global().GetValue(getConfigKey(name), nil).PopulateStruct(cfg)

	reporter := &metrics.LoggingTrafficReporter{Prefix: service.Name()}

	module := &HttpModule{
		ModuleBase: *core.NewModuleBase(HTTPModuleType, name, service, reporter, roles),
		config:     *cfg,
	}
	return module, nil
}

func (m *HttpModule) Initialize(service *core.Service) error {
	return nil
}

func (m *HttpModule) AddHandlerCreateFunc(createFunc CreateHandlerFunc) {
	if m.createHandlerFuncs == nil {
		m.createHandlerFuncs = []CreateHandlerFunc{}
	}
	m.createHandlerFuncs = append(m.createHandlerFuncs, createFunc)
}

// TODO: remove

func getTypeKey(si *annotate.StructInfo) string {
	if si != nil {
		name := si.Name()
		if si.Package() != nil {
			name = si.Package().Name() + "." + name
		}
		return name
	}
	return ""
}

func (m *HttpModule) Start() chan error {
	// use annotate to discover endpoints
	// load them into the endpoint map
	handlers := annotate.
		Packages().
		Structs().
		Methods().
		WithAnnotation(HTTPHandler{})

	handlerTypes := map[string]interface{}{}

	m.router = mux.NewRouter()

	// for each of those annotations, set up the HTTP handler.
	//
	for _, v := range handlers.ToSlice() {

		httpMeta := v.AnnotationValueOfType((*HTTPHandler)(nil)).(HTTPHandler)

		var handlerInstance interface{}

		if v.Struct() != nil {
			typeKey := getTypeKey(v.Struct())
			hi, ok := handlerTypes[typeKey]

			if !ok {
				t := v.Struct().Type()
				if t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
				instance := reflect.New(t)
				hi = instance.Interface()
				handlerTypes[typeKey] = hi
			}
			handlerInstance = hi
		}

		m.setupHTTPHandler(httpMeta, v, handlerInstance)
	}

	http.Handle("/", m.router)

	// finally, start the http server.
	//
	log.Printf("Server listening on port %d\n", m.config.Port)

	ret := make(chan error, 1)

	// Set up the socket
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", m.config.Port))

	if err != nil {
		ret <- err
		return ret
	}
	m.listener = listener

	go func() {
		ret <- http.Serve(m.listener, m.router)
	}()
	return ret
}

func (m *HttpModule) Stop() error {

	// TODO: thread safety
	if m.listener != nil {
		m.listener.Close()
		m.listener = nil
	}
	return nil
}

func (m *HttpModule) IsRunning() bool {
	return false
}

// Set up the centralized HTTP Handler which:
//
// 1. Registers the handler with http
// 2. Invokes the correct method when the handler is called
// 3. Takes the result struct to specify the http result, handle any errors, etc.
// 4. Formats the body based on the specified content type.  Default is "text/plain", but we also support application/json.
// 5. Tracks the timing of the execution
// 6. Sets status code, content type, and content length for the result.
// 7. Logs the request and it's response length.
//
// TODOS:
// 1. This doesn't respect HTTP verbs, so to do that we'd need to group all of the handlers
// that we found by path, then by verb and dispatch them to the correct method based on the VERB+PATH combo
//
// 2. We'd want to add more stuff to handle panics coming from the handler so we could convert those to nice HTTP 500s instead
// of a sticky mess on the floor.
//
func (m *HttpModule) setupHTTPHandler(info HTTPHandler, handler *annotate.MethodInfo, instance interface{}) {

	// check if this handler has an HTTPContext
	//
	ctxName := ""
	ctxPos := 0

	if len(handler.Parameters()) > ctxPos {
		rt := handler.Parameters()[ctxPos].Type()
		ctxType := reflect.TypeOf((*HTTPContext)(nil))
		if rt.AssignableTo(ctxType) {
			ctxName = handler.Parameters()[ctxPos].Name()
		}
	}

	reporter := m.Reporter()
	key := fmt.Sprintf("%s_%s", info.Verb, info.Path)

	// register the handler with net/http
	m.router.HandleFunc(
		info.Path,
		func(rw http.ResponseWriter, req *http.Request) {

			// todo refactor to use standard result processing
			defer func() {
				if r := recover(); r != nil {
					rw.WriteHeader(http.StatusInternalServerError)
					rw.Header().Add(ContentType, ContentTypeText)
					bytes := []byte(fmt.Sprintf("Error calling %s %s:\n\t%v\n", info.Verb, info.Path, r))
					rw.Write(bytes)
				}
			}()

			data := map[string]string{
				metrics.TrafficCorrelationID: req.Header.Get("RequestID"),
			}
			tracker := reporter.Start(key, data, 90*time.Second)
			defer context.Clear(req)

			// start the timer and call the handler
			//
			var responseValue []interface{}
			if ctxName != "" {
				argMap := make(map[string]interface{})
				argMap[ctxName] = &HTTPContext{
					req: req,
					res: rw,
				}
				for k, v := range mux.Vars(req) {
					argMap[k] = v
				}

				for k, v := range req.URL.Query() {
					val := ""
					if len(v) > 0 {
						val = v[0]
					}
					argMap[k] = val
				}
				responseValue = handler.InvokeByName(instance, argMap, false)
			} else {
				responseValue = handler.Invoke(instance, req)
			}

			// unpack the response envelope
			response := responseValue[0].(*HTTPResponse)
			if response != nil {

				contentType := response.ContentType

				// Setup headers
				//
				if response.Headers != nil {

					if ct, ok := response.Headers[ContentType]; ok {
						contentType = ct
					}

					for n, v := range response.Headers {
						rw.Header().Set(n, v)
					}
				}

				var body string

				if contentType == "" {
					contentType = ContentTypeText
				}

				// format content type
				switch contentType {
				case ContentTypeText:
					body = fmt.Sprintf("%v", response.Body)
				case ContentTypeJSON:
					// handle JSON if specified
					if json, err := json.MarshalIndent(response.Body, "", " "); err == nil {
						body = string(json)
					} else {
						response.Status = 500
						body = fmt.Sprintf("Error: %v", err)
					}
				}

				// set headers for content type and length
				//
				rw.Header().Set(ContentType, contentType)
				bytes := []byte(body + "\n")
				rw.Header().Set(ContentLength, fmt.Sprintf("%d", len(bytes)))

				// write the actual response, log the results and timing.
				rw.WriteHeader(response.Status)
				rw.Write(bytes)
			} else {
				rw.WriteHeader(http.StatusNoContent)
				rw.Write(make([]byte, 0))
			}

			var err error
			if response != nil {
				err = response.Error
			}
			tracker.Finish("", response, err)
			//log.Printf("%v %s => %d %dμs %d bytes\n", info.Verb, info.Path, response.Status, stop.Sub(start).Nanoseconds()/1000, len(bytes))

		},
	).Methods(info.Verb)
}

const (
	ContentType     = "Content-Type"
	ContentLength   = "Content-Length"
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
)

// HTTPResponse is an envelope for returning the results of an HTTP call
//
type HTTPResponse struct {
	Status      int
	ContentType string
	Body        interface{}
	Headers     map[string]string
	Error       error
}
