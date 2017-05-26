package main

import "go.uber.org/fx"

func main() {
	app := fx.New(provide)
	app.RunForever(invoke, callThisAtStart)
}

type p struct{}

func provide() *p { return &p{} }

func invoke() {
}

func callThisAtStart() {

}
