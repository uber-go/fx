package core

type Lifecycle interface {
	Start() <-chan error
	Stop() <-chan error
}
