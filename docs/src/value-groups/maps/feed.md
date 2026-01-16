# Feeding Map Value Groups

Map value groups allow you to provide values that can be consumed as a map indexed by name,
rather than just as a slice. To enable map consumption, values must be provided with both
a `name` and a `group` tag.

## Basic Map Feeding

To feed values into a map value group, use both `name` and `group` tags in `fx.ResultTags`:

```go
--8<-- "value-groups/maps/feed.go:feed-email"
```

The key requirements are:

1. **Both tags required**: Values must have both `name:"key"` and `group:"groupname"` tags
2. **Unique names**: Each name within a group must be unique
3. **Same group**: All values intended for the same map must use the same group name

## Multiple Handlers

You can provide multiple handlers to the same group with different names:

```go
--8<-- "value-groups/maps/feed.go:feed-slack"
```

```go
--8<-- "value-groups/maps/feed.go:feed-webhook"
```

## Interface Casting

When providing concrete types that should be consumed as interfaces,
use `fx.As` to cast the type:

```go
fx.Annotate(
    NewEmailHandler,
    fx.As(new(Handler)),                    // Cast to interface
    fx.ResultTags(`name:"email" group:"handlers"`),
)
```

This ensures that consumers receive the interface type rather than the concrete implementation.

## Naming Strategy

Choose meaningful, unique names that describe the purpose or type of each handler:

- Use descriptive names: `"email"`, `"slack"`, `"webhook"`
- Avoid generic names: `"handler1"`, `"handler2"`
- Be consistent: Use a naming convention across your application

!!! tip

    The names you choose will become the keys in the map when consumed.
    Make them descriptive and meaningful to consumers.

## Decoration Restrictions

Map value groups have specific restrictions when used with decorators:

- **Slice decorations are forbidden**: You cannot decorate a map value group consumed as a slice
- **Map decorations are allowed**: You can decorate map value groups when consumed as maps
- **Mixed consumption**: If a group is consumed both as a map and slice, decoration is not permitted

This restriction exists because decorating slices within map groups would create inconsistencies
between the map and slice representations of the same group.

## Error Conditions

Map value groups will fail if:

- A value has a `group` tag but no `name` tag
- Two values in the same group have the same name
- The map consumer's key type doesn't match the name type (must be `string`)
- You attempt to decorate a slice when the group is also consumed as a map