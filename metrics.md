### Metrics Emitted by Kyma Metrics Collector:

| Metric                                                  | Description                                                                                                             |
| ------------------------------------------------------- | :---------------------------------------------------------------------------------------------------------------------- |
| **kmc_edp_request_duration_seconds**                    | Duration of HTTP request to EDP in seconds.                                                                             |
| **kmc_keb_request_duration_seconds**                    | Duration of HTTP request to KEB in seconds.                                                                             |
| **kmc_process_sub_account_total**                       | Number of sub-accounts processed including successful and failed.                                                       |
| **kmc_process_sub_account_processed_timestamp_seconds** | Unix timestamp (in seconds) of last successful processing of sub-account.                                               |
| **kmc_process_old_metric_published_gauge**              | Number of consecutive re-sends of old metrics to edp per cluster. It Will reset to 0 when new metric data is published. |
| **kmc_process_fetched_clusters**                        | All clusters fetched from KEB including trackable and not trackable.                                                    |
| **kmc_skr_query_total**                                 | Total number of queries to SKR to get the metrics of the cluster.                                                       |
