# Integration Tests
This folder contains a suite of tests to exercise the `SCANOSS GO API` service.

It contains two ways to exercise the locally deployed API:
* CURL commands
* Go test suite

## CURL Commands
The full set of curl commands can be found in [curl_commands.txt](curl_commands.txt).

## Go Test Suite
The go test suite can be reviewed in each `*_test.go` file in this folder.

They can be exercised using the following commands.

### Local Test
Test against a service already running locally (on port `5443`):
```shell
make int_test
```

# End-to-End Test
Test against the service via containers:
```shell
make e2e_test
```
