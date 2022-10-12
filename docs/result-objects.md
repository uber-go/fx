# Result Objects

A result object is an object with the sole purpose of carrying results
for a specific function or method.

As with [parameter objects](parameter-objects.md),
the object is defined exclusively for a single function,
and not shared with other functions.

In Fx, result objects contain exported fields exclusively,
and are always tagged with `fx.Out`.

**Related**

- [Parameter objects](parameter-objects.md) are the parameter analog of result
  objects.

## Using result objects

To use result objects in Fx, take the following steps:

1. Define a new struct type named after your constructor
   with a `Result` suffix.
   If the constructor is named `NewClient`, name the struct `ClientResult`.
   If the constructor is named `New`, name the struct `Result`.
   This naming isn't strictly necessary, but it's a good convention to follow.

   ```go mdox-exec='region ex/result-objects/define.go empty'
   type ClientResult struct {
   }
   ```

2. Embed `fx.Out` into this struct.

   ```go mdox-exec='region ex/result-objects/define.go fxout'
   type ClientResult struct {
     fx.Out
   ```

3. Use this new type as the return value of your constructor *by value*.

   ```go mdox-exec='region ex/result-objects/define.go returnresult'
   func NewClient() (ClientResult, error) {
   ```

4. Add values produced by your constructor as **exported** fields on this struct.

   ```go mdox-exec='region ex/result-objects/define.go fields'
   type ClientResult struct {
   	fx.Out

   	Client *Client
   }
   ```

5. Set these fields and return an instance of this struct from your
   constructor.

   ```go mdox-exec='region ex/result-objects/define.go produce'
   func NewClient() (ClientResult, error) {
   	client := &Client{
   		// ...
   	}
   	return ClientResult{Client: client}, nil
   }
   ```

<!--
TODO: cover various tags supported on a result object.
TODO: cover adding new results
-->
