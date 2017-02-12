# Generic Module

This provides functionality to implement new modules more easily.

```go
func NewFooModule(options ...module.Option) service.ModuleCreateFunc {
  return generic.NewModule("foo", &fooModule{}, &fooConfig{}, options...)
}

type fooConfig struct {
  modules.ModuleConfig
  Foo string
}

type fooModule struct {
  generic.Controller
  config *fooConfig
}

func (m *fooModule) Initialize(
  contoller generic.Controller,
  config interface{},
) error {
  m.Controller = controller
  m.config = config.(*fooConfig)
  return nil
}

func (m *fooModule) Start() error {
  return nil
}

func (m *fooModule) Stop() error {
  return nil
}
```
