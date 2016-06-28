//go:generate annotate -package main -output annotations.go

//@import uhttp "github.com/uber-go/uberfx/modules/http"
package main

import (
	"math/rand"
	"net/http"
	"time"

	uhttp "github.com/uber-go/uberfx/modules/http"
)

// Handlers is the class that holds our HTTP handlers
// @uhttp.ServiceAccess{ Allowed: "api, vehicles, populous, rt-api", Denied: "udestroy"}
type Handlers struct {
	id int
}

// @uhttp.HTTPHandler{Verb: "GET", Path:"/"}
func (h Handlers) HandleRoot(req *http.Request) *uhttp.HTTPResponse {
	return &uhttp.HTTPResponse{
		Status: 200,
		Body:   "Hello World!",
	}
}

// @uhttp.HTTPHandler{Verb: "GET", Path:"/search"}
func (h Handlers) HandleSearch(ctx *uhttp.HTTPContext, name string, city string) *uhttp.HTTPResponse {
	return &uhttp.HTTPResponse{
		Status: 200,
		Body:   "You asked for: " + name + " from " + city,
	}
}

// @uhttp.HTTPHandler{Verb: "GET", Path:"/health"}
// @uhttp.HTTPAuth{AllowAnonymous: true}
func (h Handlers) HandleHealth(req *http.Request) *uhttp.HTTPResponse {
	return &uhttp.HTTPResponse{
		Status: 200,
		Body:   ";-)",
	}
}

// @uhttp.HTTPHandler{Verb: "GET", Path:"/time"}
func (h Handlers) HandleTime(req *http.Request) *uhttp.HTTPResponse {
	return &uhttp.HTTPResponse{
		Status: 200,
		Body:   time.Now(),
	}
}

type randomStuff struct {
	Number int       `json:"num"`
	Time   time.Time `json:"time"`
}

// HandleRandom handles random stuff
// @uhttp.HTTPHandler{Verb: "GET", Path:"/random"}
// @uhttp.HTTPAuth{RequiredRoles:"admin, employee"}
func (h Handlers) HandleRandom(req *http.Request) *uhttp.HTTPResponse {
	return &uhttp.HTTPResponse{
		Status:      200,
		ContentType: uhttp.ContentTypeJSON,
		Headers:     map[string]string{"X-My-Header": "foo"},
		Body:        randomStuff{Time: time.Now(), Number: rand.Int()},
	}
}

// func init() {
// 	handler := Handlers{}

// 	routing.Register(
// 		h.HandleRandom,
// 		uhttp.HTTPHandler{
// 			Verb: "GET",
// 			Path: "/random",
// 		},
// 	)
// }
