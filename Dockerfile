FROM golang:1.19 as build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./

RUN go generate ./pkg/cmd/server.go
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o ./scanoss-wayuu2 ./cmd/server


FROM debian:buster-slim

WORKDIR /app
 
COPY --from=build /app/scanoss-wayuu2 /app/scanoss-wayuu2

EXPOSE 8085

ENTRYPOINT ["./scanoss-wayuu2"]
#CMD ["--help"]
