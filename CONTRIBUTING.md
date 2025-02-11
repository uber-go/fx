---
search:
  exclude: true
---

# Contributing

Thanks for helping to make Fx better for everyone!

If you'd like to add new exported APIs,
please [open an issue](https://github.com/uber-go/fx/issues/new)
describing your proposal.
Discussing API changes ahead of time makes pull request review much smoother.

!!! tip

    You'll need to sign [Uber's CLA](https://cla-assistant.io/uber-go/fx)
    before we can accept any of your contributions.
    If necessary, a bot will remind
    you to accept the CLA when you open your pull request.


## Contribute code

Set up your local development environment to contribute to Fx.

1. [Fork](https://github.com/uber-go/fx/fork), then clone the repository.

    === "Git"

        ```bash
        git clone https://github.com/your_github_username/fx.git
        cd fx
        git remote add upstream https://github.com/uber-go/fx.git
        git fetch upstream
        ```

    === "GitHub CLI"

        ```bash
        gh repo fork --clone uber-go/fx
        ```

2. Install Fx's dependencies:

     ```bash
     go mod download
     ```

3. Verify that tests and other checks pass locally.

     ```bash
     make lint
     make test
     ```

     Note that for `make lint` to work,
     you must be using the latest stable version of Go.
     If you're on an older version, you can still contribute your change,
     but we may discover style violations when you open the pull request.

Next, make your changes.

1. Create a new feature branch.

     ```bash
     git checkout master
     git pull
     git checkout -b cool_new_feature
     ```

2. Make your changes, and verify that all tests and lints still pass.

     ```bash
     $EDITOR app.go
     make lint
     make test
     ```

3. When you're satisfied with the change,
   push it to your fork and make a pull request.

    === "Git"

        ```bash
        git push origin cool_new_feature
        # Open a PR at https://github.com/uber-go/fx/compare
        ```

    === "GitHub CLI"

        ```bash
        gh pr create
        ```

At this point, you're waiting on us to review your changes.
We *try* to respond to issues and pull requests within a few business days,
and we may suggest some improvements or alternatives.
Once your changes are approved, one of the project maintainers will merge them.

The review process will go more smoothly if you:

- add tests for new functionality
- write a [good commit message](https://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)
- maintain backward compatibility
- follow our [style guide](https://github.com/uber-go/guide/blob/master/style.md)

## Contribute documentation

To contribute documentation to Fx,

1. Set up your local development environment
   as you would to [contribute code](#contribute-code).

2. [Install uv](https://docs.astral.sh/uv/getting-started/installation/).
   We use this to manage our Python dependencies.

3. Run the development server.

     ```bash
     make serve
     ```

4. Make your changes.

Documentation changes should adhere to the guidance laid out below.

### Document by purpose

Documentation is organized in one of the following categories.

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

Explanations and how-to guides are often on the same page,
but they should be in distinct sections.

This separation is inspired by the
[Divio documentation system](https://documentation.divio.com/),

### Formatting

#### ATX-style headers

Use ATX-style headers (`#`-prefixed),
not Setext-style (underlined with `===` or `---`).

```markdown
Bad header
==========

## Good header
```

#### Semantic Line Breaks

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

### Test everything

All code samples in documentation must be buildable and testable.

To make this possible, we put code samples in the "ex/" directory,
and use the [PyMdown Snippets extension](https://facelessuser.github.io/pymdown-extensions/extensions/snippets/)
to include them in the documentation.

To include code snippets in your documentation,
take the following steps:

1. Add source code under the `ex/` directory.
   Usually, the file will be placed in a directory
   with a name matching the documentation file
   that will include the snippet.

     For example,
     for src/annotation.md, examples will reside in ex/annotation/.

2. Inside the source file, name regions of code with comments in the forms:

     ```
     // --8<-- [start:name]
     ...
     // --8<-- [end:name]
     ```

     Where `name` is the name of the snippet.
     For example:

     ```go
     // --8<-- [start:New]
     func New() *http.Server {
         // ...
     }
     // --8<-- [end:New]
     ```

3. Include the snippet in a code block with the following syntax:

    ```markdown
    ```go
    ;--8<-- "path/to/file.go:name"
    ```

    Where `path/to/file.go` is the path to the file containing the snippet
    relative to the `ex/` directory,
    and `name` is the name of the snippet.

    You can include multiple snippets from the same file like so:

    ```
    ;--8<-- "path/to/file.go:snippet1"
    ;--8<-- "path/to/file.go:snippet2"
    ```
