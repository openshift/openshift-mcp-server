## v0.140.0

### 🛑 Breaking changes 🛑

- `all`: Latest supported k8s version is moved from 1.17 to 1.21. (#43891)
- `cmd/otelcontribcol`: Removing unmaintained component extension/ecstaskobserver (#43818)
- `exporter/elasticsearch`: Upgrade profiles proto to 1.9.0 (#44031)
- `extension/googlecloudlogentry_encoding`: Parse cloud armor logs into log record attributes instead of placing it in the body as is. (#43389)
- `pkg/ottl`: Upgrade profiles proto to 1.9.0 (#44031)
- `processor/filter`: Upgrade profiles proto to 1.9.0 (#44031)
- `processor/tail_sampling`: Replace policy latency metric with total time spent executing specific sampling policy. (#42620)
  The existing latency metric was misleading and expensive to compute. The new cpu time metric can be used to find expensive policies instead.
- `receiver/pprof`: Upgrade profiles proto to 1.9.0 (#44031)
- `receiver/prometheus`: The prometheus receiver no longer adjusts the start time of metrics by default. (#43656)
  Disable the receiver.prometheusreceiver.RemoveStartTimeAdjustment | feature gate to temporarily re-enable this functionality. Users that need | this functionality should migrate to the metricstarttime processor, | and use the true_reset strategy for equivalent behavior.

### 🚀 New components 🚀

- `pkg/translator/pprof`: move code from receiver/pprofreceiver to pkg/translator/pprof (#43975)
  pprof is an established format. For a better integration with OTel move code to a dedicated component.
- `receiver/awslambda`: Add scaffolding for the new awslambda receiver, for event-triggered receiving of data from S3 and CloudWatch. (#43504)
- `receiver/googlecloudpubsubpush`: Add skeleton to the google pub sub push receiver. (#43503)
- `receiver/systemd`: Report active state of systemd units. (#33532)
- `receiver/yanggrpc`: New component YANG gRPC (#43840)

### 💡 Enhancements 💡

- `exporter/azureblob`: Added `serial_num_enabled` and `time_parser_enabled` options to `blob_name_format` in Azure Blob Exporter to control random serial number appending and time parsing behavior. (#43603)
- `exporter/elasticsearch`: Add support for latest OTEL SemConv version and fix Elasticsearch exporter ECS mapping for message.destination.name which is different for Elastic spans or transactions (#43805, #43806)
- `exporter/elasticsearch`: Add helpful error hint for illegal_argument_exception when using OTel mapping mode with Elasticsearch < 8.12 (#39282)
  When using OTel mapping mode (default from v0.122.0) with Elasticsearch versions < 8.12,
  the exporter now provides a more descriptive error message explaining that OTel mapping mode
  requires Elasticsearch 8.12+ and suggests either upgrading Elasticsearch or using a different
  mapping mode. This helps users who encounter the generic illegal_argument_exception error
  understand the root cause and resolution steps.
  
- `exporter/googlecloudstorage`: Add googlecloudstorageexporter to the contrib distribution (#44063)
- `exporter/kafka`: Adds a new configuration option to the Kafka exporter to control the linger time for the producer. (#44075)
  Since `franz-go` now defaults to `10ms`, it's best to allow users to configure this option to suit their needs.
- `extension/datadog`: Adds collector resource attributes to collector metadata payload (#43979)
  The Collector's resource attributes can be set under `service::telemetry::resource`.
- `extension/encoding`: Add most of the AWS ELB fields to the AWSLogsEncoding. (#43757)
- `receiver/datadog`: Adding log telemetry functionality to the existing datadog receiver component. (#43841)
- `receiver/github`: Add `include_span_events` for GitHub Workflow Runs and Jobs for enhanced troubleshooting (#43180)
- `receiver/journald`: Add root_path and journalctl_path config for running journald in a chroot (#43731)
- `receiver/prometheusremotewrite`: Skip emitting empty metrics. (#44149)
- `receiver/prometheusremotewrite`: prometheusremotewrite receiver now accepts metric type unspcified histograms. (#41840)
- `receiver/redis`: Add redis metrics that are present in telegraf: cluster_enabled, tracking_total_keys, used_memory_overhead, used_memory_startup (#39859)
- `receiver/splunkenterprise`: added pagination for search cases which may return more than the default 100 results (#43608)
- `receiver/webhookevent`: Allow configuring larger webhook body size (#43544)
  The receiver allows configuration a larger body buffer if needed. 
  It also returns an error if the body exceeds the configured limit.
  

### 🧰 Bug fixes 🧰

- `cmd/opampsupervisor`: Redacts HTTP headers in debug message (#43781)
- `connector/datadog`: Datadog connector no longer stalls after a downstream component errors (#43980)
- `exporter/awsxray`: Fix conversion of the inProgress attribute into a Segment field instead of metadata (#44001)
- `exporter/datadog`: Fix a panic from a race condition between exporter shutdown and trace export (#44068)
- `exporter/elasticsearch`: Handle empty histogram buckets to not result in an invalid datapoint error. (#44022)
- `exporter/elasticsearch`: Update the ecs mode span encode to correctly encode `span.links` ids as `trace.id` and `span.id` (#44186)
- `exporter/elasticsearch`: Improve error message when an invalid Number data point is received. (#39063)
- `exporter/loadbalancing`: Ensure loadbalancing child exporters use the OTLP type so backend creation succeeds (#43950)
- `exporter/stef`: Fix STEF connection creation bug (#44048)
  On some rare occasions due to a bug STEF exporter was incorrectly disconnecting just | created STEF connection causing connection error messages in the log. This fixes the bug.
- `extension/bearertokenauth`: Remove error messages `fsnotify: can't remove non-existent watch` when watching kubernetes SA tokens. (#44104)
- `processor/k8sattributes`: The fix is on k8sattributes processor to only set k8s.pod.ip attribute when it is requested in the extract.metadata configuration. (#43862)
  Previously, the `k8s.pod.ip` attribute was always populated, even if it was not included in the `extract.metadata` list. 
  This fix ensures that `k8s.pod.ip` is set only when explicitly requested, aligning the processor behavior with configuration expectations.
  
- `receiver/ciscoos`: Rename receiver component name from `ciscoosreceiver` to `ciscoos` to follow naming conventions. (#42647)
  Users must update their collector configuration from `ciscoosreceiver/device` to `ciscoos/device`.
  This is acceptable as the component is in alpha stability.
  
- `receiver/sqlserver`: Resolved inaccurate data sampling in query metrics collection. (#44303)
- `receiver/sqlserver`: Fix incorrect logic in query metrics window calculation. (#44162)
- `receiver/sqlserver`: Fixed a bug in effective value calculation of lookback time in top query collection. (#43943)
- `receiver/windowsservice`: Fixed an error where incorrect permissions and bad error handling were causing the receiver to stop reporting metrics (#44087)

<!-- previous-version -->


## v1.46.0/v0.140.0

### 💡 Enhancements 💡

- `cmd/mdatagen`: `metadata.yaml` now supports an optional `entities` section to organize resource attributes into logical entities with identity and description attributes (#14051)
  When entities are defined, mdatagen generates `AssociateWith{EntityType}()` methods on ResourceBuilder
  that associate resources with entity types using the entity refs API. The entities section is backward
  compatible - existing metadata.yaml files without entities continue to work as before.
  
- `cmd/mdatagen`: Add semconv reference for metrics (#13920)
- `connector/forward`: Add support for Profiles to Profiles (#14092)
- `exporter/debug`: Disable sending queue by default (#14138)
  The recently added sending queue configuration in Debug exporter was enabled by default and had a problematic default size of 1.
  This change disables the sending queue by default.
  Users can enable and configure the sending queue if needed.
  
- `pkg/config/configoptional`: Mark `configoptional.AddEnabledField` as beta (#14021)
- `pkg/otelcol`: This feature has been improved and tested; secure-by-default redacts configopaque values (#12369)

### 🧰 Bug fixes 🧰

- `all`: Ensure service service.instance.id is the same for all the signals when it is autogenerated. (#14140)

<!-- previous-version -->
