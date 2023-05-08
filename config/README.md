# Sample SCANOSS Scanning GO API Configs
This folder contains some examples of configuration for running the SCANOSS GO API.

There are two types of configuration:
* Application Config
* IP Filtering

## App Config
There are two configs provided here:
* Dev - [app-config-dev.json](app-config-dev.json)
* Prod - [app-config-prod.json](app-config-prod.json)

A description of each field and its intended usage can be found in [server_config.go](../pkg/config/server_config.go).

## IP Filtering
There are two types of IP filtering supports:
* Allow List - [allow_list.txt](allow_list.txt)
* Deny List - [deny_list.txt](deny_list.txt)

The implementation for this is based on [jpillora/ipfilter](https://github.com/jpillora/ipfilter).

Configuration for this is controlled via the `Filtering` block in the [config file](app-config-prod.json).

Currently, specific IP addresses and subnet masks are supported. Blocking by default can be controlled via `Filtering -> BlockByDefault` and Proxy support using `Filtering -> TrustProxy`.

## Detailed ZAP Logging Config
There is an optional ZAP configuration file in this folder also:
* [zap-logging-prod.json](zap-logging-prod.json)

Details for configuring this file can be found [here](https://pkg.go.dev/go.uber.org/zap) and the exact structure can be found in [config.go](https://github.com/uber-go/zap/blob/master/config.go).

To add this config to the startup please add the `ConfigFile` option to the `Logging` section of the configuration:
```json
{
  "Logging": {
    "ConfigFile": "config/zap-logging-prod.json"
  }
}
```
