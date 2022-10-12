# Parameter Objects

A parameter object is an objects with the sole purpose of carrying parameters
for a specific function or method.

The object is typically defined exclusively for that function,
and is not shared with other functions.
That is, a parameter object is not a general purpose object like "user"
but purpose-built, like "parameters for the `GetUser` function".

In Fx, parameter objects consist of exported fields exclusively,
and are always tagged with `fx.In`.

## Using parameter objects

To use parameter objects in Fx, take the following steps:

1. Define a new struct type named after your constructor.
   If the constructor is named `NewClient`, name the struct `ClientParams`.
   If the constructor is named `New`, name the struct `Params`.
   This naming isn't strictly necessary, but it's a good convention to follow.

   ```go mdox-exec='region ex/parameter-objects/define.go empty'
   type ClientParams struct {
   }
   ```

2. Embed `fx.In` into this struct.

   ```go mdox-exec='region ex/parameter-objects/define.go fxin'
   type ClientParams struct {
     fx.In
   ```

3. Add this new type as a parameter to your constructor *by value*.

   ```go mdox-exec='region ex/parameter-objects/define.go takeparam'
   func NewClient(p ClientParams) (*Client, error) {
   ```

4. Add dependencies of your constructor as **exported** fields on this struct.

   ```go mdox-exec='region ex/parameter-objects/define.go fields'
   type ClientParams struct {
   	fx.In

   	Config     ClientConfig
   	HTTPClient *http.Client
   }
   ```

5. Consume these fields in your constructor.

   ```go mdox-exec='region ex/parameter-objects/define.go consume'
   func NewClient(p ClientParams) (*Client, error) {
     return &Client{
       url:  p.Config.URL,
       http: p.HTTPClient,
       // ...
     }, nil
   ```

<!--
TODO: cover various tags supported on a parameter object.
-->
