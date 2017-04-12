# Configuration Package

At a high level, configuration is any data that is used in an application but
not part of the application itself. Any reasonably complex system needs to
have knobs to tune, and not everything can have intelligent defaults.

In UberFx, we try very hard to make configuring UberFx convenient. Users can:

* Get components working with minimal configuration
* Override any field if the default doesn't make sense for their use case

## Nesting

The configuration system wraps a set of _providers_ that each know how to get
values from an underlying source:

* Static YAML configuration
* Command line flags

So by stacking these providers, we can have a priority system for defining
configuration that can be overridden by higher priority providers. For example,
the static YAML configuration would be the lowest priority and those values
should be overridden by values specified as environment variables.

As an example, imagine a YAML config that looks like:

```yaml
foo:
  bar:
    boo: 1
    baz: hello

stuff:
  server:
    port: 8081
    greeting: Hello There!
```

UberFx Config allows direct key access, such as `foo.bar.baz`:

```go
cfg := svc.Config()
if value := cfg.Get("foo.bar.baz"); value.HasValue() {
  fmt.Println("Say", value.AsString()) // "Say hello"
}
```

Or via a strongly typed structure, even as a nest value, such as:

```go
type myStuff struct {
  Port     int    `yaml:"port" default:"8080"`
  Greeting string `yaml:"greeting"`
}

// ....

target := &myStuff{}
cfg := svc.Config()
if err := cfg.Get("stuff.server").Populate(target); err != nil {
  // fail, we didn't find it.
}

fmt.Println("Port is", target.Port) // "Port is 8081"
```

This model respects priority of providers to allow overriding of individual
values. Read `Loader` section for more details about loader process.

## Provider

`Provider` is the interface for anything that can provide values.
We provide a few reference implementations (environment and YAML), but you are
free to register your own providers via `config.RegisterProviders()` and
`config.RegisterDynamicProviders`.

### Static configuration providers

Static configuration providers conform to the `Provider` interface
and are bootstrapped first. Use these for simple providers such as file-backed or
environment-based configuration providers.

### Dynamic Configuration Providers

Dynamic configuration providers frequently need some bootstrap configuration to
be useful, so UberFx treats them specially. Dynamic configuration providers
conform to the `Provider` interface, but they're instantiated
**after** the Static `Provider`s on order to read bootstrap values.

For example, if you were to implement a ZooKeeper-backed
`Provider`, you'd likely need to specify (via YAML or environment
variables) where your ZooKeeper nodes live.

## Value

`Value` is the return type of every configuration providers'
`Get(key string)` method. Under the hood, we use the empty interface
(`interface{}`) since we don't necessarily know the structure of your
configuration ahead of time.

You can use a `Value` for two main purposes:

* Get a single value out of configuration.

For example, if we have a YAML configuration like so:

```yaml
one:
  two: hello
```

You could access the value using "dotted notation":

```go
foo := provider.Get("one.two").AsString()
fmt.Println(foo)
// Output: hello
```

To get an access to the root element use `config.Root`:

```go
root := provider.Get(config.Root).AsString()
fmt.Println(root)
// Output: map[one:map[two:hello]]
```

* Populate a struct (`Populate(&myStruct)`)

The `As*` method has two variants: `TryAs*` and `As*`. The former is a
two-value return, similar to a type assertion, where the user checks if the second
`bool` is true before using the value.

The `As*` methods are similar to the `Must*` pattern in the standard library.
If the underlying value cannot be converted to the requested type, `As*` will
`panic`.

## Populate

`Populate` is akin to `json.Unmarshal()` in that it takes a pointer to a
custom struct and fills in the fields. It returns a `true` if the requested
fields were found and populated properly, and `false` otherwise.

For example, say we have the following YAML file:

```yaml
hello:
  world: yes
  number: 42
```

We could deserialize into our custom type with the following code:

```go
type myConfig struct {
  World  string
  Number int
}

m := myConfig{}
provider.Get("hello").Populate(&m)
fmt.Println(m.World)
// Output: yes
```

Note that any fields you wish to deserialize into must be exported, just like
`json.Unmarshal` and friends.

### Benchmarks

Current performance benchmark data:

```
BenchmarkYAMLCreateSingleFile-8                    117 allocs/op
BenchmarkYAMLCreateMultiFile-8                     204 allocs/op
BenchmarkYAMLSimpleGetLevel1-8                       0 allocs/op
BenchmarkYAMLSimpleGetLevel3-8                       0 allocs/op
BenchmarkYAMLSimpleGetLevel7-8                       0 allocs/op
BenchmarkYAMLPopulate-8                             18 allocs/op
BenchmarkYAMLPopulateNested-8                       42 allocs/op
BenchmarkYAMLPopulateNestedMultipleFiles-8          52 allocs/op
BenchmarkYAMLPopulateNestedTextUnmarshaler-8       233 allocs/op
BenchmarkZapConfigLoad-8                           136 allocs/op
```

## Environment variables

YAML provider supports accepting values from the environment.
For example, consider the following YAML file:

```yaml
modules:
  http:
    port: ${HTTP_PORT:3001}
```

Upon loading file, YAML provider will look up the HTTP_PORT environment variable
and if available use it's value. If it's not found, the provided `3001` default
will be used.

## Command line arguments

Command line provider is a static provider that reads flags passed to a program and
wraps them in the `Provider` interface. It uses dots in flag names as separators
for nested values and commas to indicate that flag value is an array of values.
For example:

```go
type Wonka struct {
  Source string
  Array  []string
}

type Willy struct {
  Name Wonka
}

func main() {
  pflag.CommandLine.String("Name.Source", "default value", "String example")
  pflag.CommandLine.Var(
    &config.StringSlice{},
    "Name.Array",
    "Example of a nested array")

  var v Willy
  config.DefaultLoader().Load().Get(config.Root).Populate(&v)
  log.Println(v)
}
```

If you run this program with arguments
`./main --Name.Source=chocolateFactory --Name.Array=cookie,candy`, it will print
`{{chocolateFactory [cookie candy]}}`

## Testing

`Provider` interface makes unit testing easy, you can use the config that was
loaded with service or mock it with a static provider. For example, lets create
a calculator type, that does operations with 2 arguments:

```go
// Operation is a simple binary function.
type Operation func(left, right int) int

// Calculator evaluates operation Op on its Left and Right fields.
type Calculator struct {
  Left  int
  Right int
  Op    Operation
}

func (c Calculator) Eval() int {
  return c.Op(c.Left, c.Right)
}
```

Calculator constructor needs only `config.Provider` and it loads configuration from
the root:

```go
func NewCalculator(cfg Provider) (*Calculator, error){
  calc := &Calculator{}
  return calc, cfg.Get(Root).Populate(calc)
}
```

`Operation` has a  function type, but we can make it configurable. In order for
a provider to know how to deserialize it, `Operation` type needs to implement
`text.Unmarshaller` interface:

```go
func (o *Operation) UnmarshalText(text []byte) error {
  switch s := string(text); s {
  case "+":
    *o = func(left, right int) int { return left + right }
  case "-":
    *o = func(left, right int) int { return left - right }
  default:
    return fmt.Errorf("unknown operation %q", s)
  }

  return nil
}
```

Testing it with a static provider will be easy, we can define all arguments there
with the expected result:

```go
func TestCalculator_Eval(t *testing.T) {
  t.Parallel()

  table := map[string]Provider{
    "1+2": NewStaticProvider(map[string]string{
      "Op": "+", "Left": "1", "Right": "2", "Expected": "3"}),
    "1-2": NewStaticProvider(map[string]string{
      "Op": "-", "Left": "2", "Right": "1", "Expected": "1"}),
  }

  for name, cfg := range table {
    t.Run(name, func(t *testing.T) {
      calc, err := NewCalculator(cfg)
      require.NoError(t, err)
      assert.Equal(t, cfg.Get("Expected").AsInt(), calc.Eval())
    })
  }
}
```

We should not forget to test the error path as well:

```go
func TestCalculator_Errors(t *testing.T) {
  t.Parallel()

  _, err := newCalculator(NewStaticProvider(map[string]string{
    "Op": "*", "Left": "3", "Right": "5"
  }))

  require.Error(t, err)
  assert.Contains(t, err.Error(), `unknown operation "*"`)
}
```

For integration/E2E testing you can customize `config.Loader` to load
configuration files from either custom folders(`Loader.SetDirs()`),
or custom files(`Loader.SetFiles()`), or register new providers on top of
existing providers(`Loader.RegisterProviders()`) that will override values
of default configs.

## Utilities

`Config` package comes with several helpers that can make writing tests,
create new providers or amend existing ones much easier.

* `NewCachedProvider(p Provider)` returns a new provider that wraps `p`
  and caches values in underlying map. It also registers callbacks to track
  changes in all values it cached, so you can call `cached.Get("something")`
  and don't worry about latencies much. It is safe for concurrent use by
  multiple goroutines.

* `MockDynamicProvider` is a mock provider that can be used to test dynamic
  features, it implements `Provider` interface and lets you to set values
  to trigger change callbacks.

* Sometimes dynamic providers let you to register only one callback per key.
  If you want to have multiple keys per callback you can use
  `NewMultiCallbackProvider(p Provider)` wrapper, that will store a list of
  all callbacks for each value and call them when a value changes.
  Caution: it locks provider during callbacks execution, you should try to
  make this callbacks as fast as possible.

* `NopProvider` is useful for testing because it can be embedded in any type
  if you are not interested in implementing all Provider methods.

* `NewProviderGroup(name string, providers ...Provider)` groups providers in one.
  Lookups for values are determined by the order providers passed:
  `NewProviderGroup("global", provider1, provider2)`, first `provider1` will be
  checked and if there is no value, it will return `provider2.Get()`.

* `NewStaticProvider(data interface{})` is very a useful wrapper for testing,
  you can pass custom maps and use them as configs instead of loading them
  from files.

## Loader

Load process is controlled by `config.Loader`. If a service doesn't specify a
config provider, manager is going to use a provider returned by
`config.DefaultLoader.Load()`.

The default loader will load static providers first:

* YAML provider will look for `base.yaml` and `${environment}.yaml` files in
  current folder and then in `./config` folder. You can override folders
  to look for these files with `Loader.SetDirs()`.
  To override files names use `Loader.SetFiles()`.

* Command line provider will look for `--roles` argument to specify service
  roles. You can introduce/override config values by adding new flags to
  `pflags.CommandLine` set before building a service.

You can add more static providers on top of mentioned above with
`RegisterProviders()` function:

```go
config.DefaultLoader().RegisterProviders(
  func() Provider, error {
    return config.NewStaticProvider(map[string]int{"1+2": 3})
  }
)
```

After static providers are loaded, they are used to create dynamic providers.
You can add new ones in the loader with `RegusterDynamicProviders()` call as well.

In the end all providers are grouped together using
`NewProviderGroup("globa", staticProviders, dynamicProviders)` and returned to service.

If all you want is just a config, there is no need to build a service, you can use
`config.DefaultLoader.Load()` and get exactly the same config.

Loader type is very customizable and lets you write parallel tests easily: if you
don't want to use `os.LookupEnv()` function to look for environment variables you
can override it with your custom function: `config.DefaultLoader.SetLookupFn()`.
