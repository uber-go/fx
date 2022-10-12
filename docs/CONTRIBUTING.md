# Contributing to Fx Documentation

If you'd like to contribute to Fx's documentation, read this first.

## Document by purpose

Documentation in this folder should fall in one of the following categories.

- **Tutorials**: These hold step-by-step instructions for an end-to-end project
  that a beginner could follow along to.
  Don't spend time explaining things.
  If explanations are available elsewhere, link to them.
  These are entry points to answer the prompt,
  "I don't know what Fx is, show me what it can do,"
  so there won't be too many of these.
- **Explanations**: These hold long-form explanations of concepts and ideas.
  These are intended to build an understanding of Fx.
  Feel free to go wild here--use learning aids like diagrams, tables, etc.
- **How-tos**: These are step-by-step instructions for a *specific problem*.
  Unlike tutorials, these are not meant to be end-to-end.
  Feel free to leave things out, make assumptions,
  or provide options ("if you're doing this, do this").
  As with tutorials, don't spend time explaining;
  link to explanations elsewhere.

As an example,

- A tutorial will use lifecycle hooks as part of
  a larger set of instructions for a full end-to-end application.
- An explanation will explain what lifecycle hooks are, how they work,
  when and how you should use them, and link to relevant APIs and guides.
- A how-to guide will demonstrate how to use lifecycle hooks
  with an HTTP server, a gRPC server, etc.

This separation is inspired by the
[Divio documentation system](https://documentation.divio.com/),
which suggests separating documentation into four categories:
the three above, and references, which we get from our generated API reference.

## Formatting

### GitHub Flavored Markdown

The documentation is hosted on GitHub,
and therefore it follows [GitHub Flavored Markdown](https://github.github.com/gfm/).

### ATX-style headers

Use ATX-style headers (`#`-prefixed),
not Setext-style (underlined with `===` or `---`).

```markdown
Bad header
==========

# Good header
```

### Semantic Line Breaks

- **Do not** write overly long lines of text
- **Do not** "reflow" Markdown paragraphs
- **Do** use [Semantic Line Breaks](https://sembr.org/) to break these lines down

```markdown
This is a bad paragraph because it's really long, all on one line. When I open this in a text editor, I'll have to scroll right.

This is a bad paragraph because even though it's not all one one line, it adds
line breaks when it reaches the line length limit. This means that anytime I
change anything in this paragraph, I have to "reflow" it, which will change
other lines and make the change I'm making more difficult to review.

This is a good paragraph. It uses semantic line breaks.
I can add words or modify an existing sentence,
or even parts of a sentence,
easily and without affecting other lines.
When I change something, the actual change I made is easy to review.
Markdown will reflow this into a "normal" pargraph when rendering.
```

## Test everything

All code samples in documentation must be buildable and testable.

To aid in this, we have two tools:

- [mdox] lets us ensure that the contents of a code block are up-to-date
- the `region` shell script allows us to extract parts of a code sample

### mdox

mdox is a Markdown file formatter that includes support for
running a command and using its output as part of a code block.
To use this, declare a regular code block and tag it with `mdoc-exec`.

```markdown
```go mdox-exec="cat foo.go"
// doesn't matter
```

The contents of the code block will be replaced
with the output of the command when you run `make fmt`.
`make check` will ensure that the contents are up-to-date.

The command runs inside the directory where the Markdown file resides.
All paths should be relative to that directory.

### region

The `region` shell script is a command intended to be used with `mdox-exec`.

```plain mdox-exec="region" mdox-expect-exit-code="1"
USAGE: region FILE REGION1 REGION2 ...

Extracts text from FILE marked by "// region" blocks.
```

For example, given the file:

```
foo
// region myregion
bar
// endregion myregion
baz
```

Running `region $FILE myregion` will print:

```
bar
```

The same region name may be used multiple times
to pull different snippets from the same file.
For example, given the file:

```go
// region provide-foo
func main() {
	fx.New(
		fx.Provide(
			NewFoo,
			// endregion provide-foo
			NewBar,
		// region provide-foo
		),
	).Run()
}

// endregion provide-foo
```

`region $FILE provide-foo` will print,

```go
func main() {
	fx.New(
		fx.Provide(
			NewFoo,
		),
	).Run()
}
```
