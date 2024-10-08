# Kyma Metrics Collector

## Status

[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/kyma-metrics-collector)](https://api.reuse.software/info/github.com/kyma-project/kyma-metrics-collector)

![GitHub tag checks state](https://img.shields.io/github/checks-status/kyma-project/kyma-metrics-collector/main?label=kyma-metrics-collector&link=https%3A%2F%2Fgithub.com%2Fkyma-project%2Fkyma-metrics-collector%2Fcommits%2Fmain)

## Overview
Kyma Metrics Collector (KMC) is a component that scrapes all Kyma clusters to generate metrics. These metrics are sent to an SAP-internal tool called Event Data Platform (EDP) as an event stream and used for billing information.

Learn more about functionality and architecture in the [Contributor README](./docs/contributor/README.md).

## Usage

### Flags

Kyma Metrics Collector comes with the following command line argument flags:

| Flag | Description | Default Value   |
| ----- | ------------ | ------------- |
| `scrape-interval` | The time interval to wait between 2 executions of metrics generation. | `3m`         |
| `worker-pool-size` | The number of workers in the pool. | `5` |
| `log-level` | The log-level of the Application. For example, `fatal`, `error`, `info`, `debug`. | `info` |
| `listen-addr` | The Application starts the server in this port to cater to the metrics and health endpoints. | `8080` |
| `debug-port` | The custom port to debug when needed. `0` will disable the debugging server. | `0` |

### Environment variables

Kyma Metrics Collector comes with the following environment variables:

 | Variable | Description | Default Value   |
 | ----- | ------------ | ------------- |
 | `PUBLIC_CLOUD_SPECS` | This specification contains the CPU, Network and Disk information for all machine types from a public cloud provider.  | `-` |
 | `KEB_URL` | The KEB URL where Kyma Metrics Collector fetches runtime information. | `-` |
 | `KEB_TIMEOUT` | This timeout governs the connections from Kyma Metrics Collector to KEB | `30s` |
 | `KEB_RETRY_COUNT` | The number of retries Kyma Metrics Collector will do when connecting to KEB fails. | 5 |
 | `KEB_POLL_WAIT_DURATION` | The time interval for Kyma Metrics Collector to wait between each execution of polling KEB for runtime information. | `10m` |
 | `EDP_URL` | The EDP base URL where Kyma Metrics Collector will ingest the event-stream to. | `-` |
 | `EDP_TOKEN` | The token used to connect to EDP. | `-` |
 | `EDP_NAMESPACE` | The namespace in EDP where Kyma Metrics Collector will ingest the event-stream to.| `kyma-dev` |
 | `EDP_DATASTREAM_NAME` | The datastream in EDP where Kyma Metrics Collector will ingest the event-stream to. | `consumption-metrics` |
 | `EDP_DATASTREAM_VERSION` | The datastream version which Kyma Metrics Collector will use. | `1` |
 | `EDP_DATASTREAM_ENV` | The datastream environment which Kyma Metrics Collector will use.  | `dev` |
 | `EDP_TIMEOUT` | The timeout for Kyma Metrics Collector connections to EDP. | `30s` |
 | `EDP_RETRY` | The number of retries for Kyma Metrics Collector connections to EDP. | `3` |

## Development
- Run a deployment in a currently configured k8s cluster:
  >**NOTE:** In order to do this, you need a token from a secret `kcp-kyma-metrics-collector`.
  ```
  ko apply -f dev/
  ```

- Run tests:
  ```
  make test
  ```

### Troubleshooting
- Check logs:
  ```
  kubectl logs -f -n kcp-system $(kubectl get po -n kcp-system -l 'app=kmc-dev' -oname) kmc-dev
  ```

## Contributing

See the [Contributing Rules](CONTRIBUTING.md).

## Code of Conduct

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing

See the [license](./LICENSE) file.
