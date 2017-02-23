# Generic Module

This provides functionality to implement new modules more easily.

```go
func NewFooModule(options ...module.Option) service.ModuleCreateFunc {
  return generic.NewModule("foo", &fooModule{}, options...)
}

type fooConfig struct {
  modules.ModuleConfig
  Foo string
}

type fooModule struct {
  generic.Controller
  config fooConfig
}

func (m *fooModule) Initialize(contoller generic.Controller) error {
  m.Controller = controller
  return generic.PopulateStruct(controller, &m.config)
}

func (m *fooModule) Start() error {
  return nil
}

func (m *fooModule) Stop() error {
  return nil
}
```
