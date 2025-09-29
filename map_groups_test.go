// Copyright (c) 2023 Uber Technologies, Inc.
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

package fx_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

// Test types for map value groups
type mapTestLogger interface {
	Log(message string) string
	Name() string
}

type mapTestConsoleLogger struct {
	name string
}

func (c *mapTestConsoleLogger) Log(message string) string {
	return fmt.Sprintf("console[%s]: %s", c.name, message)
}

func (c *mapTestConsoleLogger) Name() string {
	return c.name
}

type mapTestFileLogger struct {
	name string
}

func (f *mapTestFileLogger) Log(message string) string {
	return fmt.Sprintf("file[%s]: %s", f.name, message)
}

func (f *mapTestFileLogger) Name() string {
	return f.name
}

type testHandler interface {
	Handle(data string) string
}

type testEmailHandler struct{}

func (e *testEmailHandler) Handle(data string) string {
	return "email: " + data
}

type testSlackHandler struct{}

func (s *testSlackHandler) Handle(data string) string {
	return "slack: " + data
}

type testService struct {
	name string
}

type testConfig struct {
	Key   string
	Value int
}

type testHTTPHandler struct {
	id string
}

func (h *testHTTPHandler) Process(data string) string {
	return fmt.Sprintf("http[%s]: %s", h.id, data)
}

func (h *testHTTPHandler) ID() string {
	return h.id
}

type testSimpleService interface {
	GetName() string
}

type testBasicService struct {
	name string
}

func (b *testBasicService) GetName() string {
	return b.name
}

// TestMapValueGroups tests the new map value groups functionality from dig PR #381
func TestMapValueGroups(t *testing.T) {
	t.Parallel()

	t.Run("basic map consumption", func(t *testing.T) {
		t.Parallel()

		type Params struct {
			fx.In
			// NEW: Map consumption - indexed by name
			LoggerMap map[string]mapTestLogger `group:"loggers"`
			// EXISTING: Slice consumption still works
			LoggerSlice []mapTestLogger `group:"loggers"`
		}

		var params Params
		app := fxtest.New(t,
			fx.Provide(
				// Provide loggers with BOTH name AND group
				fx.Annotate(
					func() mapTestLogger { return &mapTestConsoleLogger{name: "console"} },
					fx.ResultTags(`name:"console" group:"loggers"`),
				),
				fx.Annotate(
					func() mapTestLogger { return &mapTestFileLogger{name: "file"} },
					fx.ResultTags(`name:"file" group:"loggers"`),
				),
			),
			fx.Populate(&params),
		)
		defer app.RequireStart().RequireStop()

		// Test map consumption
		require.Len(t, params.LoggerMap, 2)
		assert.Contains(t, params.LoggerMap, "console")
		assert.Contains(t, params.LoggerMap, "file")

		consoleLogger := params.LoggerMap["console"]
		require.NotNil(t, consoleLogger)
		assert.Equal(t, "console", consoleLogger.Name())

		fileLogger := params.LoggerMap["file"]
		require.NotNil(t, fileLogger)
		assert.Equal(t, "file", fileLogger.Name())

		// Test slice consumption still works
		require.Len(t, params.LoggerSlice, 2)
		loggerNames := make([]string, len(params.LoggerSlice))
		for i, logger := range params.LoggerSlice {
			loggerNames[i] = logger.Name()
		}
		assert.ElementsMatch(t, []string{"console", "file"}, loggerNames)
	})

	t.Run("map consumption with interfaces", func(t *testing.T) {
		t.Parallel()

		type HandlerParams struct {
			fx.In
			Handlers map[string]testHandler `group:"handlers"`
		}

		var params HandlerParams
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(
					func() testHandler { return &testEmailHandler{} },
					fx.ResultTags(`name:"email" group:"handlers"`),
				),
				fx.Annotate(
					func() testHandler { return &testSlackHandler{} },
					fx.ResultTags(`name:"slack" group:"handlers"`),
				),
			),
			fx.Populate(&params),
		)
		defer app.RequireStart().RequireStop()

		require.Len(t, params.Handlers, 2)
		assert.Equal(t, "email: test", params.Handlers["email"].Handle("test"))
		assert.Equal(t, "slack: test", params.Handlers["slack"].Handle("test"))
	})

	t.Run("map consumption with pointer types", func(t *testing.T) {
		t.Parallel()

		type ServiceParams struct {
			fx.In
			Services map[string]*testService `group:"services"`
		}

		var params ServiceParams
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(
					func() *testService { return &testService{name: "auth"} },
					fx.ResultTags(`name:"auth" group:"services"`),
				),
				fx.Annotate(
					func() *testService { return &testService{name: "billing"} },
					fx.ResultTags(`name:"billing" group:"services"`),
				),
			),
			fx.Populate(&params),
		)
		defer app.RequireStart().RequireStop()

		require.Len(t, params.Services, 2)
		assert.Equal(t, "auth", params.Services["auth"].name)
		assert.Equal(t, "billing", params.Services["billing"].name)
	})

	t.Run("empty map when no providers", func(t *testing.T) {
		t.Parallel()

		type EmptyParams struct {
			fx.In
			Empty map[string]mapTestLogger `group:"empty"`
		}

		var params EmptyParams
		app := fxtest.New(t,
			fx.Populate(&params),
		)
		defer app.RequireStart().RequireStop()

		require.NotNil(t, params.Empty)
		assert.Len(t, params.Empty, 0)
	})

	t.Run("value groups cannot be optional", func(t *testing.T) {
		t.Parallel()

		type OptionalParams struct {
			fx.In
			OptionalLoggers map[string]mapTestLogger `group:"optional_loggers" optional:"true"`
		}

		var params OptionalParams
		app := NewForTest(t,
			fx.Populate(&params),
		)

		// Should fail because value groups cannot be optional
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value groups cannot be optional")
	})

	t.Run("value type maps", func(t *testing.T) {
		t.Parallel()

		type ConfigParams struct {
			fx.In
			Configs map[string]testConfig `group:"configs"`
		}

		var params ConfigParams
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(
					func() testConfig { return testConfig{Key: "db", Value: 100} },
					fx.ResultTags(`name:"database" group:"configs"`),
				),
				fx.Annotate(
					func() testConfig { return testConfig{Key: "cache", Value: 200} },
					fx.ResultTags(`name:"cache" group:"configs"`),
				),
			),
			fx.Populate(&params),
		)
		defer app.RequireStart().RequireStop()

		require.Len(t, params.Configs, 2)
		assert.Equal(t, "db", params.Configs["database"].Key)
		assert.Equal(t, 100, params.Configs["database"].Value)
		assert.Equal(t, "cache", params.Configs["cache"].Key)
		assert.Equal(t, 200, params.Configs["cache"].Value)
	})
}

// TestMapValueGroupsWithNameAndGroup tests the ability to use both dig.Name() and dig.Group()
func TestMapValueGroupsWithNameAndGroup(t *testing.T) {
	t.Parallel()

	type testHandlerWithID interface {
		Process(data string) string
		ID() string
	}

	t.Run("combine name and group annotations", func(t *testing.T) {
		t.Parallel()

		type CombinedParams struct {
			fx.In
			// Map consumption from group
			AllHandlers map[string]testHandlerWithID `group:"handlers"`
			// Individual named access
			PrimaryHandler   testHandlerWithID `name:"primary"`
			SecondaryHandler testHandlerWithID `name:"secondary"`
		}

		var params CombinedParams
		app := fxtest.New(t,
			fx.Provide(
				// This was impossible before dig PR #381!
				// Now we can use BOTH name AND group
				fx.Annotate(
					func() testHandlerWithID { return &testHTTPHandler{id: "primary"} },
					fx.ResultTags(`name:"primary" group:"handlers"`),
				),
				fx.Annotate(
					func() testHandlerWithID { return &testHTTPHandler{id: "secondary"} },
					fx.ResultTags(`name:"secondary" group:"handlers"`),
				),
			),
			fx.Populate(&params),
		)
		defer app.RequireStart().RequireStop()

		// Test map access
		require.Len(t, params.AllHandlers, 2)
		assert.Contains(t, params.AllHandlers, "primary")
		assert.Contains(t, params.AllHandlers, "secondary")

		// Test individual named access
		require.NotNil(t, params.PrimaryHandler)
		assert.Equal(t, "primary", params.PrimaryHandler.ID())
		require.NotNil(t, params.SecondaryHandler)
		assert.Equal(t, "secondary", params.SecondaryHandler.ID())

		// Verify they're the same instances
		assert.Equal(t, params.PrimaryHandler, params.AllHandlers["primary"])
		assert.Equal(t, params.SecondaryHandler, params.AllHandlers["secondary"])
	})

	t.Run("invoke function with both map and named dependencies", func(t *testing.T) {
		t.Parallel()

		var invokeCalled bool
		var mapSize int
		var primaryID string

		app := fxtest.New(t,
			fx.Provide(
				fx.Annotate(
					func() testHandlerWithID { return &testHTTPHandler{id: "main"} },
					fx.ResultTags(`name:"main" group:"handlers"`),
				),
				fx.Annotate(
					func() testHandlerWithID { return &testHTTPHandler{id: "backup"} },
					fx.ResultTags(`name:"backup" group:"handlers"`),
				),
			),
			fx.Invoke(fx.Annotate(
				func(handlers map[string]testHandlerWithID, main testHandlerWithID) {
					invokeCalled = true
					mapSize = len(handlers)
					primaryID = main.ID()
				},
				fx.ParamTags(`group:"handlers"`, `name:"main"`),
			)),
		)
		defer app.RequireStart().RequireStop()

		assert.True(t, invokeCalled)
		assert.Equal(t, 2, mapSize)
		assert.Equal(t, "main", primaryID)
	})
}

// TestMapValueGroupsEdgeCases tests edge cases and error conditions
func TestMapValueGroupsEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("duplicate names should fail", func(t *testing.T) {
		t.Parallel()

		type DuplicateParams struct {
			fx.In
			Services map[string]testSimpleService `group:"services"`
		}

		var params DuplicateParams
		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(
					func() testSimpleService { return &testBasicService{name: "first"} },
					fx.ResultTags(`name:"duplicate" group:"services"`),
				),
				fx.Annotate(
					func() testSimpleService { return &testBasicService{name: "second"} },
					fx.ResultTags(`name:"duplicate" group:"services"`),
				),
			),
			fx.Populate(&params),
		)

		// Should fail because duplicate names are not allowed
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already provided")
	})

	t.Run("map groups require all entries to have names", func(t *testing.T) {
		t.Parallel()

		type MixedParams struct {
			fx.In
			Services     map[string]testSimpleService `group:"mixed"`
			ServiceSlice []testSimpleService           `group:"mixed"`
		}

		var params MixedParams
		app := NewForTest(t,
			fx.Provide(
				// Named service
				fx.Annotate(
					func() testSimpleService { return &testBasicService{name: "named"} },
					fx.ResultTags(`name:"named_service" group:"mixed"`),
				),
				// Unnamed service - this should cause an error when map is requested
				fx.Annotate(
					func() testSimpleService { return &testBasicService{name: "unnamed"} },
					fx.ResultTags(`group:"mixed"`),
				),
			),
			fx.Populate(&params),
		)

		// Should fail because map value groups require all entries to have names
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "every entry in a map value groups must have a name")
	})

	t.Run("invalid map key types should fail", func(t *testing.T) {
		t.Parallel()

		type InvalidKeyParams struct {
			fx.In
			Services map[int]testSimpleService `group:"services"`
		}

		var params InvalidKeyParams
		app := NewForTest(t,
			fx.Provide(
				fx.Annotate(
					func() testSimpleService { return &testBasicService{name: "test"} },
					fx.ResultTags(`name:"test" group:"services"`),
				),
			),
			fx.Populate(&params),
		)

		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value groups may be consumed as slices or string-keyed maps only")
	})
}