# SCANOSS Scanning API in GO
Welcome to the SCANOSS platform. This repository serves up the REST API supporting all the scanning capabilities written in golang.

**Warning** Work In Progress **Warning**

## Repository Structure
This repository is made up of the following components:
* ?

## Configuration

Environmental variables are fed in this order:

dot-env --> env.json -->  Actual Environment Variable

Here are some of the supported configuration arguments:

```
APP_DEBUG="true"
APP_TRACE="true"
SCAN_BINARY="./tests/scanoss.sh"
LOG_DYNAMIC="true"
SCAN_DEBUG="true"
```

The most up-to-date can be found in [server_config.go](pkg/config/server_config.go).

## Docker Environment

The scanning api server can be deployed as a Docker container.

Adjust configurations by updating an .env file in the root of this repository.

**TODO** Need to add the `scanoss` binary to this image.

### How to build

You can build your own image of the SCANOSS API Server with the ```docker build``` command as follows.

```bash
make ghcr_build
```

### How to run

Run the SCANOSS API Server Docker image by specifying the environmental file to be used with the ```--env-file``` argument. 

You may also need to expose the ```APP_PORT``` on a given ```interface:port``` with the ```-p``` argument.

```bash
docker run -it -v "$(pwd)":"$(pwd)" -p 5443:5443 ghcr.io/scanoss/scanoss-api-go -json-config $(pwd)/config/app-config-docker-local-dev.json -debug
```

## Development

To run locally on your desktop, please use the following command:

```shell
go run cmd/server/main.go -json-config config/app-config-dev.json -debug
```
Note, this will simulate the `scanoss` command, so you might need to change this if you have the actual binary on your system.

After changing a dependency version, please run the following command:
```shell
go mod tidy -compat=1.19
```

## Bugs/Features
To request features or alert about bugs, please do so [here](https://github.com/scanoss/api-go/issues).

## Changelog
Details of major changes to the library can be found in [CHANGELOG.md](CHANGELOG.md).
