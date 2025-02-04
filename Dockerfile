FROM golang:1.22 as build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go generate ./pkg/cmd/server.go
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./scanoss-go-api ./cmd/server

FROM build as test

COPY test-support/scanoss.sh /app/scanoss.sh

FROM debian:buster-slim as production

WORKDIR /app
 
COPY --from=build /app/scanoss-go-api /app/scanoss-go-api

EXPOSE 5443

ENTRYPOINT ["./scanoss-go-api"]
#CMD ["--help"]
