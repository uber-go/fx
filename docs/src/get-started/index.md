# Get started with Fx


This introduces you to the basics of Fx.
In this tutorial you will:

- [start an empty application](minimal.md)
- [add an HTTP server to it](http-server.md)
- [register a handler with the server](echo-handler.md)
- [add logging to your application](logger.md)
- [refactor to loosen coupling to your handler](registration.md)
- [add another handler to the server](another-handler.md)
- [generalize your implementation](many-handlers.md)

First, get set up for the rest of the tutorial.

1. Start a new empty project.

   ```bash
   mkdir fxdemo
   cd fxdemo
   go mod init example.com/fxdemo
   ```

2. Install the latest version of Fx.

   ```bash
   go get go.uber.org/fx@latest
   ```

Now begin by [creating a minimal application](minimal.md).
