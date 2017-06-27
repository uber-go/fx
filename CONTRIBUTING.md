# Developing UberFx

This doc is intended for contributors to UberFx (hopefully that's you!).

## Development environment

* **Go**. Install on OS X with `brew install go`. Make sure `go version` returns at
  least `1.7` since we're going to be using 1.7+ features like subtests.

* **[overcommit](https://github.com/brigade/overcommit)**, a git hook manager.
  Install `overcommit` into your path with `sudo gem install overcommit`.
  Enable it on the UberFx repo with `overcommit --install`.
  We use overcommit to enforce a variety of style, semantic, and legal things
  (for example, license headers on all source files).

## Checking out the code

Make sure the repository is cloned to the correct location:

```bash
go get go.uber.org/fx/...
cd $GOPATH/src/go.uber.org/fx
```

## Dependency management

Dependencies are tracked via `glide.yaml`. If you're not familiar with `glide`,
[read the docs](https://github.com/Masterminds/glide#usage).

## License headers

This project is open source software and requires a header at the beginning of
every source file. This is enforced by commit hooks and TravisCI.

To add license headers, use
[uber-licence](https://github.com/uber/uber-licence):

```lang=bash
make add-uber-licence
```
Note that `uber-licence` is spelled with two c's for historical reasons.

## Commit Messages

Overcommit adds some requirements to your commit messages. At Uber, we follow the
[Chris Beams](http://chris.beams.io/posts/git-commit/) guide to writing Git
commit messages. Read it, follow it, learn it, love it.

## FIXMEs

If you ever are in the middle of writing code and remember a thing you need to
do, leave a comment like:

```go
// FIXME(ai) make this actually work
```

Your initials in parens are optional but good practice. This is better
than a TODO because our CI checks for unresolved FIXMEs. If you forget to fix
a FIXME, your code won't get merged.

## Testing

Run all the tests with coverage and race detector enabled:

```bash
make test RACE=-race
```

### Disabling the race detector

The race detector makes the tests run way slower. To disable it:

```bash
make test
```

### Docker

You can run the same steps that we do for continuous integration:

```bash
make dockerci
```

If this passes, you can expect continuous integration to pass.

TODO(gs) come up with something better for this.

### Viewing HTML coverage

```bash
make coverage.html && open coverage.html
```

You'll need to have [gocov-html](https://github.com/matm/gocov-html) installed:

```bash
go get -u gopkg.in/matm/v1/gocov-html
```

## Package documentation

UberFx uses [md-to-godoc](https://github.com/sectioneight/md-to-godoc) to
generate `doc.go` package documentation from `README.md` Markdown syntax. This
means that all package-level documentation is viewable both on GitHub and
[godoc.org](https://godoc.org/go.uber.org/fx).

To document a new package, create a `README.md` in the package directory.
Once you're satisfied with its contents, run `make gendoc` from the root of the
project to rebuild the `doc.go` files.

Note that changes to documentation may take a while to propagate to godoc.org.
If you check in a change to package documentation, you can manually trigger a
refresh by scrolling to the bottom of the page on godoc.org and clicking
"Refresh Now."
