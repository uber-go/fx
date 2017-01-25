// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package auth is the Auth Package.
//
// Use package auth for request authentication and authorization for user-to-service
// and service-to-service communication. Use
// package auth for scenarios that require stricter
// validations and usage restrictions.
// auth.Client included in the package, provides seamless
// integration with presently written modules in the service framework. As a middleware,
// package authprovides additional restrictions on who can access a service, and who can be
// authenticated to access the service.
// The
// package auth doesn't dictate how authentication and authorization should work, nor which
// algorithm the security service should use. It allows client integration with the service framework.
//
//
// Auth calls
//
// SetAttribute:
// SetAttribute sets necessary request attributes for authentication. By setting attributes, security service can
// identify the service and user as well as grant certificate for further access.
//
//
// Authentication:
// Access the authentication API by calling an entity to authenticate itself. The authenticate call
// returns context, which must be populated by the backend service with signed certificate that is valid for a time frame.
//
//
// Authorization:
// Access the authorization API by the service entity to authorize its callers. Context provided by a
// request must have a signed certificate, which the caller received on authentication.
//
//
// Integrating custom auth service
//
// package auth just provides an interface and API integration with existing modules. Users can define
// their own backend security framework and integrate its clients with the service framework by following simple steps:
//
//
// • Implement auth.Client interface for custom security service
//
// Example implementation of userAuthClient:
//
//   _ Client = &userAuthClient{}
//
//   type userAuthClient struct {
//     // embed backend security service client here
//   }
//
//   func userAuthClient(info CreateAuthInfo) auth.Client {
//   	return &userAuthClient{}
//   }
//
//   func (*userAuthClient) Name() string {
//   	return "userAuthClient"
//   }
//
// • Implement custom auth APIs with auth.Client by delegating calls to your service's client
//
//   func (u *userAuthClient) Authenticate(ctx context.Context) (context.Context, error) {
//     // authenticate with backend security server
//     ctx, err := u.Client.Authenticate(ctx)
//   	return ctx, err
//   }
//
//   func (u *userAuthClient) Authorize(ctx context.Context) error {
//     // authorize with backend security server
//     err := u.Client.Authorize(ctx)
//   	return err
//   }
//
// • Register custom implementation construct with fx
//
// The last step is to integrate the user auth client with the framework. This can be done by implementing init
// function and registering the client with
// fx.
//
//   func init() {
//     auth.RegisterClient(userAuthClient)
//   }
//
//
package auth
