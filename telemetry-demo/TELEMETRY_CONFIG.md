# SCANOSS API Telemetry Configuration Guide

This guide explains how to enable and configure telemetry in any SCANOSS API deployment.

> **🚀 Demo**: For a working example with pre-configured telemetry, see the [Telemetry Demo](./README.md)

## When to Use This Guide

Use this guide when you need to:
- Enable telemetry in a production SCANOSS API instance
- Configure custom OTEL exporters
- Understand available metrics
- Integrate with your existing observability stack

## Quick Setup

**⚠️ [Telemetry is disabled by default](https://github.com/scanoss/api.go/blob/23f2b03a5a65f3e9dec9be6d4d50f1b23aa69848/pkg/config/server_config.go#L117)**, you must enable it to collect metrics.

### Option 1: Configuration File (Recommended)

Edit your service configuration file and change:
```json
"Telemetry": {
  "Enabled": false,    // ← Change to true
  "ExtraMetrics": false, // ← Change to true  
  "OltpExporter": "localhost:4317"
}
```
### Option 2: Environment Variables
```bash
export OTEL_ENABLED=true
export OTEL_EXTRA=true  
export OTEL_EXPORTER_OLTP=localhost:4317
```

### Option 3: Dot ENV File
Create `.env` file in project root:
```bash
OTEL_ENABLED=true
OTEL_EXTRA=true
OTEL_EXPORTER_OLTP=localhost:4317
```

**Configuration Loading Order**: `dot-env --> env.json --> Actual Environment Variable` (see [Configuration](../README.md#configuration) in README.md for details)

## Available Metrics

The following metrics are exposed by the SCANOSS API (defined in [utils_service.go](https://github.com/scanoss/api.go/blob/main/pkg/service/utils_service.go)):

| Metric Name | Type | Description 
|------------|------|-------------
| `scanoss-api.scan.file_count` | Counter | Files received per scan request 
| `scanoss-api.scan.file_size` | Counter | Total bytes scanned 
| `scanoss-api.contents.req_count` | Counter | File contents requests 
| `scanoss-api.license.req_count` | Counter | License details requests 
| `scanoss-api.attribution.req_count` | Counter | Attribution requests 
| `scanoss-api.scan.req_time` | Histogram | Scan duration (ms) 
| `scanoss-api.scan.file_time` | Histogram | Per-file scan time (ms) 
| `scanoss-api.scan.req_time_sec` | Histogram | Scan duration (seconds) 
| `scanoss-api.scan.file_time_sec` | Histogram | Per-file scan time (seconds) 

## Metric Name Translation

When metrics are exported to Prometheus via OTEL Collector, the names are automatically translated:

| OTEL Name (in code) | Prometheus Name (in queries) |
|---------------------|------------------------------|
| `scanoss-api.scan.file_count` | `scanoss_api_scan_file_count_total` |
| `scanoss-api.scan.file_size` | `scanoss_api_scan_file_size_total` |


## Testing Your Configuration

Once telemetry is enabled, verify it's working:

1. **Generate real scan metrics using scanoss-py**:
   ```bash
   # Install the official Python client
   pip install scanoss
   
   # Scan a file/directory to generate metrics (replace YOUR_API_URL)
   scanoss-py scan . --api-url YOUR_API_URL
   
   # Or scan a specific file
   scanoss-py scan myfile.py --api-url YOUR_API_URL
   ```

2. **Monitor OTEL Collector connection**:
   Look for connection logs in your API output

3. **Query in your metrics system**:
   - Prometheus: Start typing "scanoss" for autocomplete
   - Other systems: Check their specific query syntax

> **Note**: Replace `YOUR_API_URL` with your actual SCANOSS API endpoint (e.g., `https://api.example.com` or `http://localhost:5443`)

