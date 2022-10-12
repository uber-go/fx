# Create a minimal application

Let's build the hello-world equivalent of an Fx application.
This application won't do anything yet except print a bunch of logs.

3. Write a minimal `main.go`.

   ```go mdox-exec='region ex/get-started/01-minimal/main.go main'
   package main

   import "go.uber.org/fx"

   func main() {
   	fx.New().Run()
   }
   ```

4. Run the application.

   ```bash
   go run .
   ```

   You'll see output similar to the following.

   ```
   [Fx] PROVIDE    fx.Lifecycle <= go.uber.org/fx.New.func1()
   [Fx] PROVIDE    fx.Shutdowner <= go.uber.org/fx.(*App).shutdowner-fm()
   [Fx] PROVIDE    fx.DotGraph <= go.uber.org/fx.(*App).dotGraph-fm()
   [Fx] RUNNING
   ```

   This shows the default objects provided to the Fx application,
   but it doesn't do anything meaningful yet.
   Stop the application with `Ctrl-C`.

   ```
   [Fx] RUNNING
   ^C
   [Fx] INTERRUPT
   ```

**What did we just do?**

We build an empty Fx application by calling `fx.New` with no arguments.
Applications will normally pass arguments to `fx.New` to set up their
components.

We then run this application with the `App.Run` method.
This method blocks until it receives a signal to stop,
and it then runs any cleanup operations necessary before exiting.

Fx is primarily intended for long-running server applications;
these applications typically receive a signal from the deployment system
when it's time to shut down.
