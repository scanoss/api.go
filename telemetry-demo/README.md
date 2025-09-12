# SCANOSS API Telemetry Demo

This demo provides a complete, pre-configured telemetry stack to visualize SCANOSS API metrics using OpenTelemetry, Prometheus, and Grafana.

## What's Included

✅ **SCANOSS API** with telemetry pre-enabled  
✅ **OpenTelemetry Collector** to receive and export metrics  
✅ **Prometheus** to store time-series data  
✅ **Grafana** for advanced dashboards  
✅ **Test SCANOSS engine** (no real scanner needed)

## Quick Start

1. **Start the complete observability stack**:
   ```bash
   cd telemetry-demo
   docker compose up --build
   ```

2. **Wait for services to start** (about 30-60 seconds)

3. **Generate metrics by making API requests**:
   
   ```bash
   # Scan every 0.5 seconds - let it run to see metrics accumulate
   watch -n 0.5 "echo 'file=056f9b95f439d915bd3d81ceee9ccf9a,1234,test.js' | \
     curl -s -X POST 'http://localhost:5443/scan/direct' \
     -F 'file=@-;filename=test.wfp'"
   ```


## Where to See Metrics

### Option 1: Prometheus UI (Built-in, Simple)
- **URL**: http://localhost:9090
- **Usage**: 
  1. Click "Graph" tab
  2. Try queries like:
     - `scanoss_api_scan_file_count_total` (total files scanned)
     - `rate(scanoss_api_scan_file_count_total[5m])` (requests per second)

### Option 2: Grafana (Advanced Dashboards)
- **URL**: http://localhost:3000
- **Login**: admin / admin
- **Setup**:
  1. Connections -> Add New Connections → Prometheus
  2. URL: `http://prometheus:9090`
  3. Create dashboard with your metrics

## Available Metrics to Query

Try these queries in Prometheus:

```promql
# Total API requests over time
scanoss_api_scan_file_count_total

# Request rate (requests per second)
rate(scanoss_api_scan_file_count_total[5m])

# Files scanned in last hour  
increase(scanoss_api_scan_file_count_total[1h])

# Total bytes scanned
scanoss_api_scan_file_size_total

# License requests
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

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Orchestrates the complete telemetry stack |
| `otel-collector-config.yml` | Configures OpenTelemetry Collector pipelines |
| `prometheus.yml` | Defines Prometheus scrape targets |
| `config/app-config-demo.json` | API configuration with telemetry enabled |
| `TELEMETRY_CONFIG.md` | Production telemetry configuration guide |
