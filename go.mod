module scanoss.com/wayuu2

go 1.19

require (
	github.com/golobby/config/v3 v3.3.1
	github.com/gorilla/mux v1.8.0
	github.com/scanoss/zap-logging-helper v0.0.1
	google.golang.org/grpc v1.43.0
)

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/golobby/cast v1.3.0 // indirect
	github.com/golobby/dotenv v1.3.1 // indirect
	github.com/golobby/env/v2 v2.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Details of how to use the "replace" command for local development
// https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive
// ie. replace github.com/scanoss/papi => ../papi
// require github.com/scanoss/papi v0.0.0-unpublished
