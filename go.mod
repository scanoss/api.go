module scanoss.com/go-api

go 1.19

require (
	github.com/go-co-op/gocron v1.33.1
	github.com/golobby/config/v3 v3.4.2
	github.com/google/uuid v1.3.1
	github.com/gorilla/mux v1.8.0
	github.com/jpillora/ipfilter v1.2.9
	github.com/scanoss/zap-logging-helper v0.2.0
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.43.0
	go.opentelemetry.io/otel v1.17.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.40.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.17.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.17.0
	go.opentelemetry.io/otel/metric v1.17.0
	go.opentelemetry.io/otel/sdk v1.17.0
	go.opentelemetry.io/otel/sdk/metric v0.40.0
	go.opentelemetry.io/otel/trace v1.17.0
	go.uber.org/zap v1.26.0
)

require (
	github.com/BurntSushi/toml v1.3.2 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golobby/cast v1.3.3 // indirect
	github.com/golobby/dotenv v1.3.2 // indirect
	github.com/golobby/env/v2 v2.2.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.17.1 // indirect
	github.com/phuslu/iploc v1.0.20230630 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/tomasen/realip v0.0.0-20180522021738-f0c99a92ddce // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.40.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/grpc v1.57.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Details of how to use the "replace" command for local development
// https://github.com/golang/go/wiki/Modules#when-should-i-use-the-replace-directive
// ie. replace github.com/scanoss/papi => ../papi
// require github.com/scanoss/papi v0.0.0-unpublished
