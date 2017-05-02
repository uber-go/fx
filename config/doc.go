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

// Package config is the Configuration Package.
//
// At a high level, configuration is any data that is used in an application but
// not part of the application itself. Any reasonably complex system needs to
// have knobs to tune, and not everything can have intelligent defaults.
//
//
// In UberFx, we try very hard to make configuring UberFx convenient. Users can:
//
// • Get components working with minimal configuration
//
// • Override any field if the default doesn't make sense for their use case
//
// Nesting
//
// The configuration system wraps a set of *providers* that each know how to get
// values from an underlying source:
//
//
// • Static YAML configuration
//
// • Command-line flags
//
// So by stacking these providers, we can have a priority system for defining
// configuration that can be overridden by higher priority providers. For example,
// the static YAML configuration would be the lowest priority and those values
// should be overridden by values specified as environment variables.
//
//
// As an example, imagine a YAML config that looks like:
//
//   foo:
//     bar:
//       boo: 1
//       baz: hello
//
//   stuff:
//     server:
//       port: 8081
//       greeting: Hello There!
//
// UberFx Config allows direct key access, such as foo.bar.baz:
//
//   cfg := svc.Config()
//   if value := cfg.Get("foo.bar.baz"); value.HasValue() {
//     fmt.Println("Say", value.AsString()) // "Say hello"
//   }
//
// Or via a strongly typed structure, even as a nest value, such as:
//
//   type myStuff struct {
//     Port     int    `yaml:"port" default:"8080"`
//     Greeting string `yaml:"greeting"`
//   }
//
//   // ....
//
//   target := &myStuff{}
//   cfg := svc.Config()
//   if err := cfg.Get("stuff.server").Populate(target); err != nil {
//     // fail, we didn't find it.
//   }
//
//   fmt.Println("Port is", target.Port) // "Port is 8081"
//
// This model respects priority of providers to allow overriding of individual
// values. Read
// Loading Configuration (#Loading-Configuration) section for more details
// about the loading process.
//
//
// Provider
//
// Provider is the interface for anything that can provide values.
// We provide a few reference implementations (environment and YAML), but you are
// free to register your own providers via
// RegisterProviders() and
// RegisterDynamicProviders().
//
// Static configuration providers
//
// Static configuration providers conform to the Provider interface
// and are bootstrapped first. Use these for simple providers such as file-backed or
// environment-based configuration providers.
//
//
// Dynamic configuration providers
//
// Dynamic configuration providers frequently need some bootstrap configuration to
// be useful, so UberFx treats them specially. Dynamic configuration providers
// conform to the
// Provider interface, but they're instantiated
// **after** the Static Providers on order to read bootstrap values.
//
// For example, if you were to implement a ZooKeeper-backed
// Provider, you'd likely need to specify (via YAML or environment
// variables) where your ZooKeeper nodes live.
//
//
// Value
//
// Value is the return type of every configuration providers'
// Get(key string) method. Under the hood, we use the empty interface
// (
// interface{}) since we don't necessarily know the structure of your
// configuration ahead of time.
//
//
// You can use a Value for two main purposes:
//
// • Get a single value out of configuration.
//
// For example, if we have a YAML configuration like so:
//
//   one:
//     two: hello
//
// You could access the value using "dotted notation":
//
//   foo := provider.Get("one.two").AsString()
//   fmt.Println(foo)
//   // Output: hello
//
// To get an access to the root element use Root:
//
//   root := provider.Get(config.Root).AsString()
//   fmt.Println(root)
//   // Output: map[one:map[two:hello]]
//
// • Populate a struct (Populate(&myStruct))
//
// The As* method has two variants: TryAs* and As*. The former is a
// two-value return, similar to a type assertion, where the user checks if the second
// bool is true before using the value.
//
// The As* methods are similar to the Must* pattern in the standard library.
// If the underlying value cannot be converted to the requested type,
// As* will
// panic.
//
// Populate
//
// Populate is akin to json.Unmarshal() in that it takes a pointer to a
// custom struct or any other type and fills in the fields. It returns an error,
// if the requested fields were not populated properly.
//
//
// For example, say we have the following YAML file:
//
//   hello:
//     world: yes
//     number: 42
//
// We could deserialize into our custom type with the following code:
//
//   type myConfig struct {
//     World  string
//     Number int
//   }
//
//   m := myConfig{}
//   provider.Get("hello").Populate(&m)
//   fmt.Println(m.World)
//   // Output: yes
//
// Note that any fields you wish to deserialize into must be exported, just like
// json.Unmarshal and friends.
//
// Environment variables
//
// The YAML provider supports accepting values from the environment in which the process
// runs. For example, consider the following YAML file:
//
//
//   modules:
//     http:
//       port: ${HTTP_PORT:3001}
//
// When it loads the file, the YAML provider looks up the HTTP_PORT environment
// variable and checks for a value to use. If the YAML provider doesn't find a value,
// it uses the provided 3001 default.
//
//
// Command-line arguments
//
// The command-line provider is a static provider that reads flags passed to a
// program and wraps them in the
// Provider interface. Dots in flag names act
// as separators for nested values (read about dotted notation in the
// Dynamic configuration providers (Dynamic-configuration-providers) section above).
// Commas indicate to the provider that the flag value is an array of values.
// For example, command
// ./service --roles=actor,writer will set roles to a slice
// with two values
// []string{"actor","writer"}.
//
// Use the pflag.CommandLine global variable to define your own flags:
//
//   type Wonka struct {
//     Source string
//     Array  []string
//   }
//
//   type Willy struct {
//     Name Wonka
//   }
//
//   func main() {
//     pflag.CommandLine.String("Name.Source", "default value", "String example")
//     pflag.CommandLine.Var(
//       &config.StringSlice{},
//       "Name.Array",
//       "Example of a nested array")
//
//     var v Willy
//     config.DefaultLoader.Load().Get(config.Root).Populate(&v)
//     log.Println(v)
//   }
//
// If you run this program with arguments
// ./main --Name.Source=chocolateFactory --Name.Array=cookie,candy, it will print
// {{chocolateFactory [cookie candy]}}
//
// Testing
//
// The Provider interface makes unit testing easy. You can use the config
// that came loaded with your service or mock it with a static provider. For example,
// let's create a calculator type that does operations with two arguments:
//
//
//   // Operation is a simple binary function.
//   type Operation func(left, right int) int
//
//   // Calculator evaluates operation Op on its Left and Right fields.
//   type Calculator struct {
//     Left  int
//     Right int
//     Op    Operation
//   }
//
//   func (c Calculator) Eval() int {
//     return c.Op(c.Left, c.Right)
//   }
//
// The calculator constructor needs only Provider and it loads configuration from
// the root:
//
//
//   func NewCalculator(cfg Provider) (*Calculator, error){
//     calc := &Calculator{}
//     return calc, cfg.Get(Root).Populate(calc)
//   }
//
// Operation has a  function type, but we can make it configurable. In order for
// a provider to know how to deserialize it,
// Operation type needs to implement the
// text.Unmarshaller interface:
//
//   func (o *Operation) UnmarshalText(text []byte) error {
//     switch s := string(text); s {
//     case "+":
//       *o = func(left, right int) int { return left + right }
//     case "-":
//       *o = func(left, right int) int { return left - right }
//     default:
//       return fmt.Errorf("unknown operation %q", s)
//     }
//
//     return nil
//   }
//
// To test with a static provider will be easy, define all arguments with the
// expected results:
//
//
//   func TestCalculator_Eval(t *testing.T) {
//     t.Parallel()
//
//     table := map[string]Provider{
//       "1+2": NewStaticProvider(map[string]string{
//         "Op": "+", "Left": "1", "Right": "2", "Expected": "3"}),
//       "1-2": NewStaticProvider(map[string]string{
//         "Op": "-", "Left": "2", "Right": "1", "Expected": "1"}),
//     }
//
//     for name, cfg := range table {
//       t.Run(name, func(t *testing.T) {
//         calc, err := NewCalculator(cfg)
//         require.NoError(t, err)
//         assert.Equal(t, cfg.Get("Expected").AsInt(), calc.Eval())
//       })
//     }
//   }
//
// Don't forget to test the error path:
//
//   func TestCalculator_Errors(t *testing.T) {
//     t.Parallel()
//
//     _, err := newCalculator(NewStaticProvider(map[string]string{
//       "Op": "*", "Left": "3", "Right": "5"
//     }))
//
//     require.Error(t, err)
//     assert.Contains(t, err.Error(), "unknown operation")
//   }
//
// For integration/E2E testing you can customize Loader to load the
// configuration files from either custom folders (
// Loader.SetDirs())
// or custom files (
// Loader.SetFiles()), or you can register providers
// on top of the existing providers (
// Loader.RegisterProviders()) that will
// override values of the default configs.
//
//
// Utilities
//
// The config package comes with several helpers for writing tests, creating
// new providers, and amending existing providers.
//
//
// • NewCachedProvider(p Provider) returns a new provider that wraps pand caches values in underlying map. It also registers callbacks to track
// changes in all cached values, so you can call
// cached.Get("something")without worrying about latency. It is safe for concurrent use by
// multiple goroutines.
//
//
// • The MockDynamicProvider is a mock provider that can be used to test dynamic
// features. It implements
// Provider interface and lets you set values
// to trigger change callbacks.
//
//
// • Sometimes dynamic providers only let you register one callback per key.
// If you want to have multiple keys per callback, use the
// NewMultiCallbackProvider(p Provider) wrapper. It stores a list of
// all callbacks for each value and calls them when a value changes.
// **Caution**: provider is locked during callbacks execution, you should try to
// make the callbacks as fast as possible.
//
//
// • NopProvider is useful for testing because it can be embedded in any type
// if you are not interested in implementing all Provider methods.
//
//
// • NewProviderGroup(name string, providers ...Provider) groups providers into
// one. Lookups for values are determined by the order providers passed:
//
//
//     group := NewProviderGroup("global", provider1, provider2)
//     value := group.Get("X")
//
// The group provider checks provider1 for "X" first. If there is no value,
//   it returns the result of
// provider2.Get().
//
// • NewStaticProvider(data interface{}) is a very useful wrapper for testing.
// You can pass custom maps and use them as configs instead of loading them
// from files.
//
//
// Loading Configuration
//
// The load process is controlled by Loader. If a service doesn't
// specify a config provider,
// service.Manager is going to use a provider
// returned by
// DefaultLoader.Load().
//
// The default loader creates static providers first:
//
// • YAML provider will look for base.yaml and ${environment}.yaml files in
// the current directory and then in the
// ./config directory. You can override
// directories to look for these files with
// Loader.SetDirs().
// To override file names, use
// Loader.SetFiles().
//
// • The command-line provider looks for --roles argument to specify service
// roles. Use
// pflags.CommandLine variable to introduce or override config
// values before building a service.
//
//
// You can add more static providers on top of those mentioned above with
// RegisterProviders() function:
//
//   config.DefaultLoader.RegisterProviders(
//     func() Provider, error {
//       return config.NewStaticProvider(map[string]int{"1+2": 3})
//     }
//   )
//
// After static providers are loaded, they are used to create dynamic providers.
// You can add new dynamic providers in the loader with the
// RegisterDynamicProviders()call as well.
//
//
// In the end all providers are grouped together using
// NewProviderGroup("global", staticProviders, dynamicProviders) and returned to
// your service.
//
//
// If you only want a config, you don't need to build a service. You can use
// DefaultLoader.Load() and get exactly the same config as service.Config().
//
// The loader type is customizable, letting you write parallel tests easily. If you
// don't want to use the
// os.LookupEnv() function to look for environment variables,
// override it with your custom function:
// DefaultLoader.SetLookupFn().
//
// Benchmarks
//
// Current performance benchmark data:
//
//   BenchmarkYAMLCreateSingleFile-8                    117 allocs/op
//   BenchmarkYAMLCreateMultiFile-8                     204 allocs/op
//   BenchmarkYAMLSimpleGetLevel1-8                       0 allocs/op
//   BenchmarkYAMLSimpleGetLevel3-8                       0 allocs/op
//   BenchmarkYAMLSimpleGetLevel7-8                       0 allocs/op
//   BenchmarkYAMLPopulate-8                             18 allocs/op
//   BenchmarkYAMLPopulateNested-8                       42 allocs/op
//   BenchmarkYAMLPopulateNestedMultipleFiles-8          52 allocs/op
//   BenchmarkYAMLPopulateNestedTextUnmarshaler-8       233 allocs/op
//   BenchmarkZapConfigLoad-8                           136 allocs/op
//
//
package config
