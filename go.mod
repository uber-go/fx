module go.uber.org/fx

go 1.13

require (
	github.com/benbjohnson/clock v1.3.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/dig v1.12.0
	go.uber.org/goleak v1.1.11
	go.uber.org/multierr v1.5.0
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20210903071746-97244b99971b
)

replace go.uber.org/dig => github.com/uber-go/dig v1.13.1-0.20220106194054-29dd17211ed4
