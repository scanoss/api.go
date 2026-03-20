# SCANOSS API Telemetry Demo

This demo provides a complete, pre-configured telemetry stack to visualize SCANOSS API metrics using
OpenTelemetry, Promtail, Prometheus, Loki, Tempo, and Grafana.

## What's Included
- ✅ **SCANOSS API** with telemetry pre-enabled 
- ✅ **Test SCANOSS engine** (no real scanner needed)
- ✅ **OpenTelemetry Collector** to receive and export metrics/traces
- ✅ **Promtail Collector** to receive and export logs
- ✅ **Prometheus** to store time-series data
- ✅ **Tempo** to store spans/traces data
- ✅ **Loki** to store log data 
- ✅ **Grafana** for advanced dashboards

## Quick Start

1. **Start the complete observability stack**:
   ```bash
   cd telemetry-demo
   docker compose up --build
   ```

2. **Wait for services to start** (about 30-60 seconds)

3. **Generate metrics by making API requests**:
Run a continuous scan (every 2 seconds):
   ```bash
   # Scan every 2 seconds - let it run to see metrics accumulate
   watch -n 2.0 "echo 'file=056f9b95f439d915bd3d81ceee9ccf9a,1234,test.js' | \
     curl -s -X POST 'http://localhost:5443/scan/direct' \
     -F 'file=@-;filename=test.wfp'"
   ```
Or run a single one-off scan:
```bash
echo 'file=056f9b95f439d915bd3d81ceee9ccf9a,1234,test.js' | curl -s -X POST 'http://localhost:5443/scan/direct' -F 'file=@-;filename=test.wfp' 
```

## Visualisation
The complete observability stack is configured to export metrics, logs, and traces. This can all be visualised in Grafana.
To login to Grafana, please browse the following URL:
- http://localhost:3000
- No username/password required

From there, it is possible to explore the metrics, logs, and traces.

### Where to See Logs
To view logs, please browse the following URL:
1. http://localhost:3000/a/grafana-lokiexplore-app/explore/job/scanoss-api/logs
2. Select `job` from the "Labels" dropdown and either set equal to `scanoss-api` or leave it blank
3. And select the logs from below 
4. From here, it is possible to filter and explore the logs.

### Where to See Traces
To view traces, please browse the following URL:
1. http://localhost:3000/explore
2. Select "Tempo" from the "Outline" dropdown
3. Select one of the following Query types:
   3.1 Select `Query type` "Search" and the time window and the list of Traces below
   3.2 Select "TraceQL" from the Query type and enter a `trace_id` to search for
4. These trace IDs can then be queried in the Loki logs to see detailed information about the trace.

### Where to See Metrics
#### Option 1: Grafana (Advanced Dashboards)
1. **URL**: http://localhost:3000/a/grafana-metricsdrilldown-app/drilldown
2. Select "prometheus" from the "Data Source" dropdown
3. Type `scanoss` into the "Search metric" field to list all SCANOSS metrics
4. Select an appropriate time window

#### Option 2: Prometheus UI (Built-in, Simple)
- **URL**: http://localhost:9090
- **Usage**: 
  1. Click the "Graph" tab
  2. Try queries like:
     - `scanoss_api_scan_file_count_total` (total files scanned)
     - `rate(scanoss_api_scan_file_count_total[5m])` (requests per second)

### Data Sources
If any datasources are not automatically configured, please configure them manually:

#### Prometheus
- **Setup**:
    1. Connections -> Add New Connections → Prometheus
    2. URL: `http://prometheus:9090`

- A list of possible queries can be found in [Available Metrics to Query](#available-metrics-to-query)

#### Loki
- **Setup**:
    1. Connections -> Add New Connections → Loki
    2. URL: `http://loki:3100`

#### Tempo
- **Setup**:
    1. Connections -> Add New Connections → Tempo
    2. URL: `http://tempo:3200`


## Available Metrics to Query

Try these queries in Prometheus:
- Total API requests over time
  ```promql
  scanoss_api_scan_file_count_total
  ```
-  Request rate (requests per second)
  ```promql
  rate(scanoss_api_scan_file_count_total[5m])
  ```
- Files scanned in last hour
  ```promql
  increase(scanoss_api_scan_file_count_total[1h])
  ```
- Total bytes scanned
  ```promql
    scanoss_api_scan_file_size_total
  ```
- License requests
  ```promql
  scanoss_api_license_req_count_total
  ```

## Stop the Demo
```bash
# Stop all services
docker compose down

# Stop and remove all data
docker compose down -v
```

## Next Steps
After exploring this demo, configure telemetry for your production API using [TELEMETRY_CONFIG.md](./TELEMETRY_CONFIG.md).

## Files in This Demo

| File                          | Purpose                                      |
|-------------------------------|----------------------------------------------|
| `docker-compose.yml`          | Orchestrates the complete telemetry stack    |
| `otel-collector-config.yml`   | Configures OpenTelemetry Collector pipelines |
| `promtail-config.yml`         | Configures Promtail log Collector pipeline   |
| `loki.yaml`                   | Defines Loki logging setup                   |
| `prometheus.yml`              | Defines Prometheus scrape targets            |
| `tempo.yml`                   | Defines Tempo traces setup                   |
| `grafana-datasources.yaml`    | Data source configuration for Grafana        |
| `config/app-config-demo.json` | API configuration with telemetry enabled     |
| `TELEMETRY_CONFIG.md`         | Production telemetry configuration guide     |
