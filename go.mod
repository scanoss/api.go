module scanoss.com/go-api

go 1.19

require (
	github.com/golobby/config/v3 v3.3.1
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/scanoss/zap-logging-helper v0.0.1
)

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/golobby/cast v1.3.0 // indirect
	github.com/golobby/dotenv v1.3.1 // indirect
	github.com/golobby/env/v2 v2.2.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Details of how to use the "replace" command for local development
// https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive
// ie. replace github.com/scanoss/papi => ../papi
// require github.com/scanoss/papi v0.0.0-unpublished
