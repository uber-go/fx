module fx-map-groups-demo

go 1.25.1

require (
	go.uber.org/dig v1.19.0
	go.uber.org/fx v1.23.0
)

require (
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/sys v0.0.0-20220412211240-33da011f77ad // indirect
)

replace go.uber.org/fx => ../

replace go.uber.org/dig => github.com/jquirke/dig v0.0.0-20250929003136-0b0022552f09
