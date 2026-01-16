# Consuming Map Value Groups

Map value groups can be consumed as maps indexed by their names, providing direct access
to specific values without iteration. This is more efficient and clearer than traditional
slice consumption when you need to access specific handlers by name.

## Map Consumption

To consume a value group as a map, declare a parameter with the map type and group tag:

```go
--8<-- "value-groups/maps/consume.go:consume-map"
```

The constructor can then accept the handlers as a map:

```go
--8<-- "value-groups/maps/consume.go:new-service"
```

With map consumption, you get:

- **Direct access**: `handlers["email"]` instead of iterating through a slice
- **Type safety**: The compiler knows the key type is `string` and value type is `Handler`
- **Performance**: O(1) lookup time instead of O(n) iteration
- **Clarity**: The names make it clear what each handler does

## Mixed Consumption

You can consume the same value group as both a map and a slice:

```go
type Params struct {
    fx.In
    HandlerMap   map[string]Handler `group:"handlers"`  // Map consumption
    HandlerSlice []Handler          `group:"handlers"`  // Slice consumption
}
```

This allows different parts of your application to consume the same group in the most appropriate way.

## Slice Consumption (Traditional)

For comparison, here's the traditional slice consumption approach:

```go
--8<-- "value-groups/maps/consume.go:consume-slice"
```

```go
--8<-- "value-groups/maps/consume.go:new-service-slice"
```

With slice consumption, you lose the name information and must implement your own lookup logic.

## When to Use Maps vs Slices

**Use map consumption when:**

- You need to access specific handlers by name
- You want O(1) lookup performance
- The names provide meaningful semantic information
- You're building registries or routing systems

**Use slice consumption when:**

- You need to iterate over all values
- The order of processing matters (though value group order is not guaranteed)
- Names are not semantically meaningful
- You're migrating existing code gradually

## Map Key Types

Currently, map value groups only support `string` keys. The names you provide
in the `name` tags become the keys in the consumed map.

## Error Handling

When consuming maps, handle missing keys appropriately:

```go
func (s *NotificationService) Send(handlerType, message string) string {
    if handler, exists := s.handlers[handlerType]; exists {
        return handler.Handle(message)
    }
    return "unknown handler: " + handlerType
}
```

## Requirements

For map consumption to work:

1. **Named values**: All values in the group must have both `name` and `group` tags
2. **String keys**: Map keys must be of type `string`
3. **Unique names**: All names within a group must be unique
4. **Compatible types**: All values must be assignable to the map's value type