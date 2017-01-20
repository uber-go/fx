# Auth Package

Auth package is used for request authentication and authorization for user to service
and service to service communication. Auth is used for scenarios that require stricter
validations, and usage restrictions. `auth.Client` provided by the package provides seamless
integration with presently written modules in the service framework. As a middleware, Auth
provides additional restrictions on who can access a service, and who can be
authenticated to access the service.
The Auth package doesn't dictate how authentication and authorization should work, or which
algorithm the security service should use, but only allows client integration with the service framework.

## Auth calls
SetAttribute:
SetAttribute sets necessary request attributes for authentication. By setting attributes, security service can
identify the service/user and grant certificate for further access.

Authentication:
Authentication API is called by calling entity to authenticate itself. The Authenticate call
returns context, which must be populated by the backend service with signed certificate that is valid for a timeframe.

Authorization:
Authorization API is called by a service entity to authorize its callers. context provided by a
request must have a signed certificate which caller received on authentication.

## Integrating custom auth service
`Auth package` just provides an interface and API integration with existing modules. Users can define
their own backend security framework and integrate its clients with the service framework by following simple steps:

### Implement `auth.Client` interface for custom security service
Example implementation of userAuthClient:
```
_ Client = &userAuthClient{}

ype userAuthClient struct {
  // embed user auth client
}

func userAuthClient(info CreateAuthInfo) auth.Client {
	return &nouserAuthClientp{}
}

func (*userAuthClient) Name() string {
	return "userAuthClient"
}
```

### Impleemt custom auth APIs with `auth.Client` by delegating calls to your service's client.

### Register custom implementation construct with `fx`,
The last step is to integrate the user auth client with the framework. This can be done by implementing init
function and registering client with `fx`.
```
func init() {
  auth.RegisterClient(userAuthClient)
}
```
