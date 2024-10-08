# Prometheus P1 exporter

Prometheus exporter for smart meter statistics fetched with a P1 cable.

## Installation

With Go install:

```shell
go install github.com/loafoe/prometheus-p1-exporter@latest
```

## Usage

```
Usage of ./prometheus-p1-exporter:
  -listen string
        Listen address for HTTP metrics (default "127.0.0.1:8888")
```

The exporter will collect metrics from `/dev/ttyUSB0` every second and export the metrics to an HTTP endpoint at `http://127.0.0.1:8888/metrics`. This endpoint can be added to your Prometheus configuration.

Example metrics page:

```
# HELP p1_active_tariff Active tariff
# TYPE p1_active_tariff gauge
p1_active_tariff 2
# HELP p1_actual_electricity_delivered Actual electricity power delivered to client
# TYPE p1_actual_electricity_delivered gauge
p1_actual_electricity_delivered 0
# HELP p1_actual_electricity_generated Actual electricity power generated by client
# TYPE p1_actual_electricity_generated gauge
p1_actual_electricity_generated 0.680
# HELP p1_power_failures_long Power failures long
# TYPE p1_power_failures_long gauge
p1_power_failures_long 6
# HELP p1_power_failures_short Power failures short
# TYPE p1_power_failures_short gauge
p1_power_failures_short 13
# HELP p1_returned_electricity_high Electricity returned high tariff
# TYPE p1_returned_electricity_high gauge
p1_returned_electricity_high 835.252
# HELP p1_returned_electricity_low Electricity returned low tariff
# TYPE p1_returned_electricity_low gauge
p1_returned_electricity_low 321.717
# HELP p1_usage_electricity_high Electricity usage high tariff
# TYPE p1_usage_electricity_high gauge
p1_usage_electricity_high 25801.498
# HELP p1_usage_electricity_low Electricity usage low tariff
# TYPE p1_usage_electricity_low gauge
p1_usage_electricity_low 26137.922
# HELP p1_usage_gas Gas usage
# TYPE p1_usage_gas gauge
p1_usage_gas 0
```

## Development

You can add additional `OBISType` metrics and case statements as you see fit

## Grafana Agent

On my Rasbperry PI I'm running a copy of [Grafana Agent](https://grafana.com/docs/grafana-cloud/data-configuration/agent/) which ships the metrics to a cloud based Prometheus instance. Example config below:

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

This was forked from https://github.com/jordyv/prometheus-p1-exporter and modified for continous read outs from the P1 port
