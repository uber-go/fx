/*
Package config is the Configuration Package.

At a high level, configuration is any data that is used in an application but
not part of the application itself. Any reasonably complex system needs to
have knobs to tune, and not everything can have intelligent defaults.

In UberFx, we try very hard to make configuring UberFx convenient. Users can:

• Get components working with minimal configuration

• Override any field if the default doesn't make sense for their use case



Provider

Provider is the interface for anything that can provide values.
We provide a few reference implementations (environment and YAML), but you are
free to register your own providers via config.RegisterProviders() and
config.RegisterDynamicProviders.

Static configuration providers

Static configuration providers conform to the Provider interface
and are bootstraped first. Use these for simple providers such as file-backed or
environment-based configuration providers.

Dynamic Configuration Providers

Dynamic configuration providers frequently need some bootstrap configuration to
be useful, so UberFx treats them specially. Dynamic configuration providers
conform to the Provider interface, but they're instantianted
**after** the Static Providers on order to read bootstarp values.

For example, if you were to implement a ZooKeeper-backed
Provider, you'd likely need to specify (via YAML or environment
variables) where your ZooKeeper nodes live.

ConfigurationValue

ConfigurationValue is the return type of every configuration providers'
GetValue(key string) method. Under the hood, we use the empty interface
(interface{}) since we don't necessarily know the structure of your
configuration ahead of time.

You can use a ConfigurationValue for two main purposes:

• Get a single value out of configuration.



For example, if we have a YAML configuration like so:

  one:
    two: hello
You could access the value using "dotted notation":

  foo := provider.GetValue("one.two").AsString()
  fmt.Println(foo)
  // Output: hello
• Populate a struct (PopulateStruct(&myStruct))



The As* method has two variants: TryAs* and As*. The former is a
two-value return, similar to a type assertion, where the user checks if the second
bool is true before using the value.

The As* methods are similar to the Must* pattern in the standard library.
If the underlying value cannot be converted to the requested type, As* will
panic.

PopulateStruct

PopulateStruct is akin to json.Unmarshal() in that it takes a pointer to a
custom struct and fills in the fields. It returns a trueif the requested
fields were found and populated properly, andfalse` otherwise.

For example, say we have the following YAML file:

  hello:
    world: yes
    number: 42
We could deserialize into our custom type with the following code:

  type myConfig struct {
    World  string
    Number int
  }

  m := myConfig{}
  provider.GetValue("hello").Populate(&m)
  fmt.Println(m.World)
  // Output: yes
Note that any fields you wish to deserialize into must be exported, just like
json.Unmarshal and friends.

*/
package config
