# SCANOSS Scanning API in GO
Welcome to the SCANOSS platform. This repository serves up the REST API supporting all the scanning capabilities written in golang.

This is a replacement for the [WAYUU](https://github.com/scanoss/wayuu) and [API](https://github.com/scanoss/api) REST service/projects.

[![Unit Tests](https://github.com/scanoss/api.go/actions/workflows/go-ci.yml/badge.svg)](https://github.com/scanoss/api.go/actions/workflows/go-ci.yml)
[![Golang CI Linting](https://github.com/scanoss/api.go/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/scanoss/api.go/actions/workflows/golangci-lint.yml)

# API Usage
The API defines a number of endpoints (exact ones can be found in [server.go](pkg/protocol/rest/server.go)). The documentation for the API can be found [here](https://docs.osskb.org).

Here are some example implementations of these endpoints:
* [scanoss-py](https://github.com/scanoss/scanoss.py)
* [scanoss-js](https://github.com/scanoss/scanoss.js)


## Repository Structure
This repository is made up of the following components:
* [cmd](cmd) contains the entry point for launching the API Server
* [pkg](pkg) contains the source code to process the REST requests
* [config](config) contains sample configuration files to run the service
* [scripts](scripts) contain helper scripts for installing the service onto a Linux server

## Build and Deploy

### How to build
To build a deployable binary, please refer to the [Makefile](Makefile) for target options, including:
* build_amd
* build_arm
* package_amd
* package_arm

All of these build commands target the Linux platform and focus on either AMD64 of ARM64 architectures.
For example:
```bash
make build_amd
```
This will produce a binary in the [target](target) folder called `scanoss-go-api-amd64`. Similarly, using `build_arm` will produce a file called `scanoss-go-api-linux-arm64`.

### Packaging

Inside the [scripts](scripts) folder is a set of convenience utilities to aid deployment and running of the scanning API as a service.

For example, running the following will put a Linux AMD64 binary into the `scripts` for deployment:
```bash
make package_amd
```

### Versioning
The version of the API is calculated at compile time from the git tags (see [version.sh](version.sh)). This is injected into [server.go](pkg/cmd/server.go) at compile time.

To get the desired version in the deployed binary, please commit and tag the source before building. Standard [semantic versioning](https://semver.org) should be followed:
```bash
git tag v1.0.1
```

### Deployment
This [scripts](scripts) folder contains convenience utilities for deploying, configuring and running the SCANOSS GO API server. More details can be found [here](scripts/README.md).

### Running the Service

Once the service has been deployed on a server, it can be managed using the `systemctl` command. For example:
```bash
systemctl status scanoss-go-api
systemctl start scanoss-go-api
systemctl stop scanoss-go-api
systemctl restart scanoss-go-api
```

## Configuration

Configuration for service can be handled in three different ways:
* Dot ENV files (sample [here](.env.example))
* ENV Json files (sample [here](config/app-config-prod.json))
* Environment variables

The order in which these are loaded is as follows:

`dot-env --> env.json -->  Actual Environment Variable`

The most up-to-date configuration options can be found in [server_config.go](pkg/config/server_config.go).

## Development

### Run Local
To run locally on your desktop, please use the following command:

```shell
make run_local
```

To use a different config file, simply run the command manually using:
```shell
go run cmd/server/main.go -json-config config/app-config-dev.json -debug
```

Note, this will simulate the `scanoss` binary (using [scanoss.sh](test-support/scanoss.sh)), so you might need to change this if you have the actual binary on your system.

### Unit Testing
This project contains unit tests and can be invoked using:
```shell
make unit_test
```

### Integration Testing
This project contains integration tests and can be invoked using:
```shell
make int_test
```
This requires the service to be running locally listening on port `5443`.

It is also possible to run the whole test via containers:
```shell
make e2e_test
```

### Dependency Updates
After changing a dependency or version, please run the following command:
```shell
go mod tidy -compat=1.19
```

## Bugs/Features
To request features or alert about bugs, please do so [here](https://github.com/scanoss/api-go/issues).

## Changelog
Details of major changes to the library can be found in [CHANGELOG.md](CHANGELOG.md).
