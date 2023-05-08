module scanoss.com/go-api

go 1.19

require (
	github.com/golobby/config/v3 v3.3.1
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/jpillora/ipfilter v1.2.8
	github.com/scanoss/zap-logging-helper v0.2.0
	github.com/stretchr/testify v1.8.2
	go.uber.org/zap v1.24.0
)

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golobby/cast v1.3.0 // indirect
	github.com/golobby/dotenv v1.3.1 // indirect
	github.com/golobby/env/v2 v2.2.0 // indirect
	github.com/phuslu/iploc v1.0.20220830 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tomasen/realip v0.0.0-20180522021738-f0c99a92ddce // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Details of how to use the "replace" command for local development
// https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive
// ie. replace github.com/scanoss/papi => ../papi
// require github.com/scanoss/papi v0.0.0-unpublished
