# Parameter Objects

A parameter object is an objects with the sole purpose of carrying parameters
for a specific function or method.

The object is typically defined exclusively for that function,
and is not shared with other functions.
That is, a parameter object is not a general purpose object like "user"
but purpose-built, like "parameters for the `GetUser` function".

In Fx, parameter objects contain exported fields exclusively,
and are always tagged with `fx.In`.

**Related**

- [Result objects](result-objects.md) are the result analog of
  parameter objects.

## Using parameter objects

To use parameter objects in Fx, take the following steps:

1. Define a new struct type named after your constructor
   with a `Params` suffix.
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

## Adding new parameters

You can add new parameters for a constructor
by adding new fields to a parameter object.
For this to be backwards compatible,
the new fields must be **optional**.

1. Take an existing parameter object.

   ```go mdox-exec='region ex/parameter-objects/extend.go start'
   type Params struct {
     fx.In

     Config     ClientConfig
     HTTPClient *http.Client
   }

   func New(p Params) (*Client, error) {
   ```

2. Add a new field to it for your new dependency
   and **mark it optional** to keep this change backwards compatible.

   ```go mdox-exec='region ex/parameter-objects/extend.go full'
   type Params struct {
   	fx.In

   	Config     ClientConfig
   	HTTPClient *http.Client
   	Logger     *zap.Logger `optional:"true"`
   }
   ```

3. In your constructor, consume this field.
   Be sure to handle the case when this field is absent --
   it will take the zero value of its type in that case.

   ```go mdox-exec='region ex/parameter-objects/extend.go consume'
   func New(p Params) (*Client, error) {
     log := p.Logger
     if log == nil {
       log = zap.NewNop()
     }
     // ...
   ```
