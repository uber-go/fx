package main

import "go.uber.org/fx"

func main() {
	app := fx.New(
		provide,
	)
	app.RunForever(
		invoke,
		invoke2,
		invoke3,
		invoke4,
		invoke5,
	)
}

type p struct{}

func provide() *p { return &p{} }

func invoke() {}

func invoke2() {}

func invoke3() {}

func invoke4() {}

func invoke5(lifecycle fx.Lifecycle) {
	lifecycle.Append(fx.Hook{
		OnStart: func() error {
			return nil
		},
		OnStop: func() error {
			return nil
		},
	})
}
