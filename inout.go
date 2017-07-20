package fx

import "go.uber.org/dig"

// In can be embedded in a constructor's param struct
// in order to take advantage of named and optional types.
//
// Modules should take a single param struct that embeds an In
// in order to provide a forwards-compatible API where additional
// optional properties can be added without breaking.
type In struct{ dig.In }

// Out can be embedded in return structs in order to
// name types and provide
//
// Modules should return a single results struct that embeds
// an Out in order to provide a forwards-compatible API where
// additional types can be provided over time without breaking.
type Out struct{ dig.Out }
