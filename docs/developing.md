# Developing UberFx

This doc is intended for contributors to UberFx (hopefully that's you!)

## Testing

Run all the tests with coverage and race detector enabled:

```bash
make test
```

## Disabling the race detector

The race detector makes the tests run way slower. To disable it:

```bash
make test RACE=''
```

TODO(ai) come up with something better for this.

### Viewing HTML coverage

```bash
make coverage.html && open coverage.html
```

You'll need to have https://github.com/matm/gocov-html installed:

```bash
go get -u gopkg.in/matm/v1/gocov-html
```
