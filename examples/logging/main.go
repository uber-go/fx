package main

import "go.uber.org/fx"

func main() {
	app := fx.New(
		provide,
		provide2,
		provide3,
		provide4,
		provide5,
		provide6,
		provide7,
		provideMultiple,
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

type p2 struct{}

func provide2(lifecycle fx.Lifecycle) *p2 {
	lifecycle.Append(fx.Hook{
		OnStart: func() error {
			return nil
		},
		OnStop: func() error {
			return nil
		},
	})
	return &p2{}
}

type p3 struct{}

func provide3() *p3 { return &p3{} }

type p4 struct{}

func provide4() *p4 { return &p4{} }

type p5 struct{}

func provide5() *p5 { return &p5{} }

type p6 struct{}

func provide6() *p6 { return &p6{} }

type p7 struct{}

func provide7(p2 *p2, lifecycle fx.Lifecycle) *p7 {
	lifecycle.Append(fx.Hook{
		OnStart: func() error {
			return nil
		},
		OnStop: func() error {
			return nil
		},
	})
	return &p7{}
}

type p8 struct{}
type p9 struct{}

func provideMultiple() (*p8, *p9, error) {
	return &p8{}, &p9{}, nil
}

func invoke() {}

func invoke2() {}

func invoke3() {}

func invoke4() {}

func invoke5(p7 *p7, lifecycle fx.Lifecycle) {
	lifecycle.Append(fx.Hook{
		OnStart: func() error {
			return nil
		},
		OnStop: func() error {
			return nil
		},
	})
}
