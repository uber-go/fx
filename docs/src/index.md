# Fx

Fx is **a dependency injection system for Go**.

<div class="grid cards" markdown>

- **Eliminate globals**

    ---

    By using Fx-managed singletons,
    you can eliminate global state from your application.
    With Fx, you don't have to rely on `init()` functions for setup,
    instead relying on Fx to manage the lifecycle of your application.

- **Reduce boilerplate**

    ---

    Fx reduces the amount of code copy-pasted across your services.
    It lets you define shared application setup in a single place,
    and then reuse it across all your services.

- **Automatic plumbing**

    ---

    Fx automatically constructs your application's dependency graph.
    A component added to the application can be used by any other component
    without any additional configuration.

    [Learn more about the dependency container :material-arrow-right:](container.md)

- **Code reuse**

    ---

    Fx lets teams within your organization build loosely-coupled
    and well-integrated shareable components referred to as modules.

    [Learn more about modules :material-arrow-right:](modules.md)

- **Battle-tested**

    Fx is the backbone of nearly all Go services at Uber.

</div>

[Get started :material-arrow-right-bold:](get-started/index.md){ .md-button .md-button--primary }
