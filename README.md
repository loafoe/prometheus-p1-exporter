# Prometheus P1 exporter

Prometheus exporter for smart meter statistics fetched from a P1 meter. It supports multiple data sources, including USB Serial P1 cables and the HomeWizard P1 Meter API (v2).

## Installation

With Go install:

```shell
go install github.com/loafoe/prometheus-p1-exporter@latest
```

## Container Image

OCI images are available via **GitHub Container Registry (GHCR)**. They are multi-arch (linux/amd64, linux/arm64) and signed with `cosign`.

```shell
docker pull ghcr.io/loafoe/prometheus-p1-exporter:latest
```

Example usage with Docker:

```shell
docker run -d \
  --name p1-exporter \
  -p 8888:8888 \
  ghcr.io/loafoe/prometheus-p1-exporter:latest \
  -source=homewizard \
  -homewizard-url=https://192.168.1.100
```

## Usage

```
Usage of ./prometheus-p1-exporter:
  -listen string
        Listen address for HTTP metrics (default "127.0.0.1:8888")
  -source string
        Source of data: 'serial' (P1 USB) or 'homewizard' (HomeWizard P1 meter API) (default "serial")
  -usb-device string
        USB device for serial source (default "/dev/ttyUSB0")
  -homewizard-url string
        Base URL for HomeWizard API (e.g. https://192.168.1.100)
  -homewizard-token string
        Token for HomeWizard API (v2)
  -homewizard-token-file string
        File to store/read the HomeWizard token (default "homewizard_token.txt")
  -homewizard-interval duration
        Interval for HomeWizard API polling (default 5s)
  -verbose
        Verbose output logging
```

### Data Sources

#### USB Serial (`-source=serial`)
This is the default source. It reads directly from a P1 cable connected to your smart meter. It uses the `/dev/ttyUSB0` device by default.

#### HomeWizard P1 Meter (`-source=homewizard`)
This source polls the HomeWizard P1 Meter local API using the **v2 protocol** (HTTPS). 

**Authorization:**
The HomeWizard v2 API requires a Bearer Token. If no token is provided:
1. The exporter will initiate an authorization request.
2. It will log a warning: `Action required: Press the physical button on your HomeWizard device NOW`.
3. You have 30 seconds to **physically press the button** on your HomeWizard P1 Meter.
4. Once authorized, the exporter will save the token to `homewizard_token.txt` (or the path specified by `-homewizard-token-file`) so it can be reused across restarts.

## Metrics

The exporter provides the following metrics (prefixed with `p1_`):

| Metric | Type | Description |
| :--- | :--- | :--- |
| `usage_electricity_low` | Gauge | Electricity usage low tariff (kWh) |
| `usage_electricity_high` | Gauge | Electricity usage high tariff (kWh) |
| `returned_electricity_low` | Gauge | Electricity returned low tariff (kWh) |
| `returned_electricity_high` | Gauge | Electricity returned high tariff (kWh) |
| `actual_electricity_delivered`| Gauge | Actual electricity power delivered (kW) |
| `actual_electricity_generated`| Gauge | Actual electricity power generated (kW) |
| `active_tariff` | Gauge | Active tariff indicator |
| `power_failures_short` | Gauge | Number of short power failures |
| `power_failures_long` | Gauge | Number of long power failures |
| `usage_gas` | Gauge | Gas usage (m³) |
| `active_power_l1_w` | Gauge | Active power L1 (W) |
| `active_power_l2_w` | Gauge | Active power L2 (W) |
| `active_power_l3_w` | Gauge | Active power L3 (W) |
| `active_voltage_l1_v` | Gauge | Active voltage L1 (V) |
| `active_voltage_l2_v` | Gauge | Active voltage L2 (V) |
| `active_voltage_l3_v` | Gauge | Active voltage L3 (V) |
| `active_current_l1_a` | Gauge | Active current L1 (A) |
| `active_current_l2_a` | Gauge | Active current L2 (A) |
| `active_current_l3_a` | Gauge | Active current L3 (A) |

## Development

The project is structured with a `Source` interface, making it easy to add new data collection backends. 

```go
type Source interface {
    Start(ctx context.Context) (<-chan Reading, error)
}
```

## Grafana Agent

On my Raspberry Pi, I'm running [Grafana Agent](https://grafana.com/docs/grafana-cloud/data-configuration/agent/) which ships the metrics to a cloud-based Prometheus instance. Example config below:

```yaml
metrics:
  global:
    scrape_interval: 2s
    external_labels:
      environment: p1-server
  configs:
    - name: default
      scrape_configs:
        - job_name: 'p1_exporter'
          static_configs:
            - targets: ['localhost:8888']
      remote_write:
        - url: https://prometheus.example.com/api/v1/write
          basic_auth:
            username: scraper
            password: S0m3pAssW0rdH3Re
```

## Acknowledgement

This was forked from [jordyv/prometheus-p1-exporter](https://github.com/jordyv/prometheus-p1-exporter) and modified for continuous readouts and multiple source support.
