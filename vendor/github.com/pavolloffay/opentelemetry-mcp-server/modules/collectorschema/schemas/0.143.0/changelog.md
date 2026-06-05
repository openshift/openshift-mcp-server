## v0.143.0

### 🛑 Breaking changes 🛑

- `connector/servicegraph`: remove deprecated warning log about metrics exporter logical (#45177)
- `extension/googlecloudlogentry_encoding`: Parse Cloud DNS Query logs into log record attributes instead of placing it in the body as is. (#44561)
- `processor/tail_sampling`: Add support for caching the policy name involved in a sampling decision. (#45040)
  This change allows the `tailsampling.policy` attribute to be set on the spans in a trace when a sampling decision is cached.
- `receiver/prometheus`: Remove deprecated `use_start_time_metric` and `start_time_metric_regex` configuration options. (#44180)
  The `use_start_time_metric` and `start_time_metric_regex` configuration options have been removed after being deprecated in v0.142.0.
  Users who have these options set in their configuration will experience collector startup failures after upgrading.
  To migrate, remove these configuration options and use the `metricstarttime` processor instead for equivalent functionality.
  
- `receiver/systemd`: Rename `systemd.unit.cpu.time` metric to `systemd.service.cpu.time` (#44916)

### 🚀 New components 🚀

- `cmd/schemagen`: Introduce script that generates configuration schemas for collector components based on go structs. (#42214)
  The `schemagen` tool generates schemas for OpenTelemetry Collector components configuration
  by analyzing Go struct definitions. This is preliminary work to support automatic generation
  of documentation and validation for component configurations.
  

### 💡 Enhancements 💡

- `exporter/azureblob`: Add `time_parser_ranges` option to allow selective time parsing of blob name substrings (#44650)
  The new `time_parser_ranges` configuration allows specifying index ranges (e.g., `["0-10", "15-25"]`) to control which parts of the blob name are time-formatted.
- `exporter/coralogix`: Exposed a new field to set `grpc-accept-encoding`. `gzip` will be used by default. (#45191)
- `exporter/coralogix`: Improve log messages when a partial success happens for traces. (#44926)
  The exporter now provides additional context based on the type of partial success | returned by the backend. When the backend indicates issues with the sent data, the error | message is analyzed to identify and display examples of the problematic data.
- `exporter/elasticsearch`: Add support for extra query parameters to the outgoing bulk request (#44480)
- `exporter/kafka`: Make `max_message_bytes` and `flush_max_messages` unconditional in franz-go producer. Changed `flush_max_messages` default from 0 to 10000 to match franz-go default. (#44840)
- `extension/awslogs_encoding`: Enhance VPC flow logs encoding extension with CloudWatch logs support (#44710)
- `extension/azure_encoding`: Add processing for AppService, CDN, FrontDoor and Recommendation logs records (#41725)
- `extension/googlecloudlogentry_encoding`: Add support for Passthrough External and Internal Network Load Balancer logs (#44524)
  Add support for Passthrough External and Internal Network Load Balancer logs to the Google Cloud log entry encoding extension.
  This includes adding support for the new `gcp.load_balancing.passthrough_nlb` attributes including connection details,
  bytes/packets sent and received, and RTT measurements.
  
- `pkg/ottl`: Add `Bool` function for converting values to boolean (#44770)
- `processor/geoip`: Bump oschwald/geoip2 to v2 (#44687)
- `receiver/awscloudwatch`: Add support for filtering log groups by account ID. (#38391)
- `receiver/awscontainerinsightreceiver`: Component type name renamed from awscontainerinsightreceiver to awscontainerinsight, controlled by feature gate receiver.awscontainerinsightreceiver.useNewTypeName. (#44052)
  When the feature gate is enabled, the receiver uses the new type name `awscontainerinsight` instead of `awscontainerinsightreceiver`.
  To enable the new type name, use: `--feature-gates=+receiver.awscontainerinsightreceiver.useNewTypeName`.
  
- `receiver/awslambda`: Add support for AWS Lambda receiver to trigger by CloudWatch logs subscription filters for Lambda (#43504)
- `receiver/awslambda`: Add S3 failure replay support to AWS Lambda receiver (#43504)
- `receiver/filelog`: gzip files are auto detected based on their header (#39682)
- `receiver/github`: Add `merged_pr_lookback_days` configuration to limit historical PR queries and reduce API usage (#43388)
  The `merged_pr_lookback_days` configuration option limits the timeframe for
  which merged pull requests are queried. Set to 0 to fetch all merged PRs.
  Open PRs are always fetched regardless of this setting.
  
- `receiver/oracledb`: Add stored procedure information to logs for top queries and query samples. (#44764)
  The `db.server.top_query` event now includes `oracledb.procedure_id`, `oracledb.procedure_name`, and `oracledb.procedure_type` attributes.
  The `db.server.query_sample` event now includes `oracledb.procedure_id`, `oracledb.procedure_name`, and `oracledb.procedure_type` attributes.
  
- `receiver/postgresql`: Added `service.instance.id` resource attribute for metrics and logs (#43907)
  `service.instance.id` is enabled by default.
  
- `receiver/postgresql`: Add trace propagation support (#44868)
  When `postgresql.application_name` contains a valid W3C `traceparent`, emitted `db.server.query_sample` logs include `trace_id` and `span_id` for correlation.
  
- `receiver/prometheus`: Add `receiver.prometheusreceiver.RemoveReportExtraScrapeMetricsConfig` feature gate to disable the `report_extra_scrape_metrics` config option. (#44181)
  When enabled, the `report_extra_scrape_metrics` configuration option is ignored, and extra scrape metrics are
  controlled solely by the `receiver.prometheusreceiver.EnableReportExtraScrapeMetrics` feature gate.
  This mimics Prometheus behavior where extra scrape metrics are controlled by a feature flag.
  
- `receiver/systemd`: Add metric for number of times a service has restarted. (#45071)
- `receiver/windowseventlog`: Improved performance of the Windows Event Log Receiver (#43195)
  Previously, the Windows Event Log Receiver could only process events up to 100 messages per second with default settings.
  This was because the receiver would read at most `max_reads` messages within each configured `poll_interval`, even if
  additional events were already available.
  
  This restriction has been removed. The `poll_interval` parameter behaves as described in the documentation:
  The `poll_interval` parameter now only takes effect after all current events have been read.
  
  For users who prefer the previous behavior, a new configuration option, `max_events_per_poll`, has been introduced.
  
- `receiver/windowseventlog`: Add parsing for Version and Correlation event fields. (#45018)

### 🧰 Bug fixes 🧰

- `connector/count`: Basic config should emit default metrics (#41769)
- `exporter/elasticsearch`: Deduplicate attribute keys from non-compliant SDKs in otel mapping mode (#39304)
  The serializer in otel mapping mode now explicitly deduplicates attribute keys when writing to Elasticsearch,
  keeping only the first occurrence. This prevents invalid JSON from being produced when
  non-compliant SDKs send duplicate keys.
  
- `exporter/kafka`: Wrap non-retriable errors from franzgo with consumererror::permanent (#44918)
- `exporter/loadbalancing`: Fix k8s resolver parsing so loadbalancing exporter works with service FQDNs (#44472)
- `pkg/translator/azurelogs`: Fix missing data when ingesting Azure logs without properties field. (#44222)
- `receiver/awss3`: Fix data loss when SQS messages contain multiple S3 object notifications and some fail to process (#45153)
  The SQS notification reader was unconditionally deleting messages after processing,
  even when some S3 object retrievals or callback processing failed. This caused data
  loss when a message contained multiple S3 notification records and any of them failed.
  Messages are now only deleted when all records are successfully processed, allowing
  failed records to be retried after the visibility timeout.
  
- `receiver/azureeventhub`: Make storage of new azeventhub library backward compatible and fix checkpoint starting at earliest when storage is enabled (#44461)
- `receiver/fluentforward`: Ensure all established connections are properly closed on shutdown in the fluentforward receiver. The shutdown process now reliably closes all active connections. (#44433)
  - Fixes shutdown behavior so that all existing connections are closed cleanly.
  - Adds tests to verify proper connection closure.
  
- `receiver/kafka`: Fix deprecated field migration logic for metrics, traces, and profiles topic configuration (#45215)
  Fixed bug where deprecated `topic` and `exclude_topic` fields for metrics, traces, and profiles
  were incorrectly checking logs configuration instead of their respective signal type's configuration.
  This prevented proper migration from deprecated fields unless logs.topics was empty.
  Also fixed validation error message typo for traces.exclude_topic and corrected profiles validation
  to check ExcludeTopic fields instead of Topic fields.
  
- `receiver/sqlserver`: Collect query metrics for long running queries (#44984)
- `receiver/tcpcheck`: Fix the unit of the `tcpcheck.error` metric from `error` to `errors` (#45092)

<!-- previous-version -->


## v1.49.0/v0.143.0

### 💡 Enhancements 💡

- `all`: Update semconv import to 1.38.0 (#14305)
- `exporter/nop`: Add profiles support to nop exporter (#14331)
- `pkg/pdata`: Optimize the size and pointer bytes for pdata structs (#14339)
- `pkg/pdata`: Avoid using interfaces/oneof like style for optional fields (#14333)

<!-- previous-version -->
