module go.uber.org/fx/docs

go 1.19

require (
	github.com/stretchr/testify v1.8.0
	go.uber.org/fx v1.18.2
	go.uber.org/zap v1.23.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/dig v1.15.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/sys v0.0.0-20210903071746-97244b99971b // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.uber.org/fx => ../

replace go.uber.org/dig => github.com/uber-go/dig v1.15.1-0.20221213190814-6ef87c8bcb68
