## v0.139.0

### 🛑 Breaking changes 🛑

- `receiver/sqlserver`: Standardizing the unit interpretation of lookback_time in config for top query collection (#43573)
  Like other interval related config values, lookback_time also should suffix 's' to represent time in seconds.
  

### 💡 Enhancements 💡

- `connector/count`: Support for setting attributes from scope and resource levels. Precedence order: Span (or Log Record, etc.) > Scope attributes > Resource attributes. (#41859)
- `connector/spanmetrics`: Add `add_resource_attributes` opt-in config option to keep resource attributes in generated metrics (#43394)
  This configuration option allows users to override the `connector.spanmetrics.excludeResourceMetrics` feature gate
  and restore the old behavior of including resource attributes in metrics. This is needed for customers whose
  existing dashboards depend on resource attributes being present in the generated metrics.
  
- `exporter/azuremonitor`: Add authenticator extension support to the Azure Monitor exporter. (#41004)
- `exporter/azuremonitor`: Updated azure monitor exporter to use OTEL semantic conventions 1.34.0 (#41289)
- `exporter/datadog`: Disabled "Successfully posted payload" log that was emitted every 500 metric exports (#43594, #43879)
- `exporter/datadog`: Set sending queue batch default values to match exporter helper default: flush timeout 200ms, min size 8192, no max size. (#43848)
  The default values now match exactly the default in batch processor.
- `exporter/elasticsearch`: Update Elasticsearch exporter ECS mapping mode encoder semantic convention mappings (#43805)
- `exporter/googlecloudstorage`: Implement skeleton of googlecloudstorage exporter. (#43123)
- `exporter/influxdb`: Fix InfluxDB Exporter precision configuration to allow choice of precision instead of hardcoding 'ns'. (#43645)
- `extension/awslogs_encoding`: Enhance CloudTrail log parsing by adding support for digest files (#43403)
- `extension/awslogs_encoding`: Add support for AWS Network Firewall logs. (#43616)
  The AWS Logs Encoding Extension now supports unmarshaling AWS Network Firewall logs into OpenTelemetry logs format.
- `extension/awslogs_encoding`: Enhance CloudTrail log parsing by adding extra fields (#43403)
- `extension/googlecloudlogentry_encoding`: Add encoding.format attribute to GCP encoding extension to identify the source format. (#43320)
- `internal/aws`: Upgrade k8s libraries from v0.32.x to v0.34.x (#43890)
- `pkg/ottl`: Support taking match patterns from runtime data in the `replace_all_patterns` and `replace_pattern` functions. (#43555)
- `pkg/ottl`: Add TrimPrefix and TrimSuffix to OTTL (#43883)
  This is a much optimal way to remove prefix/suffix compare with `replace_pattern(name, "^prefixed", "")`
- `pkg/ottl`: Added support for dynamic delimiter in Split() function in OTTL. (#43555)
- `pkg/ottl`: Added support for dynamic delimiter in Concat() function in OTTL. (#43555)
- `pkg/ottl`: Added support for dynamic prefix/suffix in HasPrefix and HasSuffix functions in OTTL. (#43555)
- `pkg/ottl`: Remove unnecessary regexp compilation every execution (#43915)
- `pkg/ottl`: Add `unit` and `type` subpaths for `profile.sample_type` and `profile.period_type`. (#43723)
- `pkg/ottl`: Support taking match patterns from runtime data in the `replace_all_matches` and `replace_match` functions. (#43555)
- `pkg/ottl`: Support taking match patterns from runtime data in the `IsMatch` function. (#43555)
- `pkg/ottl`: Remove unnecessary full copy of maps/slices when setting value on sub-map (#43949)
- `pkg/ottl`: Add XXH128 Converter function to converts a `value` to a XXH128 hash/digest (#42792)
- `pkg/ottl`: Support dynamic keys in the `delete_key` and `delete_matching_keys` functions, allowing the key to be specified at runtime. (#43081)
- `pkg/ottl`: Support paths and expressions as keys in `keep_keys` and `keep_matching_keys` (#43555)
- `pkg/ottl`: Support dynamic pattern keys in `ExtractPatterns` and `ExtractGrokPatterns` functions, allowing the keys to be specified at runtime. (#43555)
- `pkg/ottl`: Added support for dynamic encoding in Decode() function in OTTL. (#43555)
- `processor/filter`: Allow setting OTTL conditions to filter out whole resources (#43968)
  If any conditions set under the `resource` key for any signals match, the resource is dropped.
- `processor/k8sattributes`: Support extracting deployment name purely from the owner reference (#42530)
- `processor/metricstarttime`: Graduate the metricstarttimeprocessor to beta. (#43656)
- `processor/redaction`: Extend database query obfuscation to span names. Previously, database query obfuscation (SQL, Redis, MongoDB) was only applied to span attributes and log bodies. Now it also redacts sensitive data in span names. (#43778)
- `processor/resourcedetection`: Add the `dt.smartscape.host` resource attribute to data enriched with the Dynatrace detector (#43650)
- `receiver/azureeventhub`: Adds support for receiving Azure app metrics from Azure Event Hubs in the azureeventhubreceiver (#41343, #41367)
  The azureeventhubreceiver now supports receiving custom metrics emitted by applications to Azure Insights and forwarded using Diagnostic Settings to Azure Event Hub.
  There's also on optional setting to aggregate received metrics into a single metric to keep the original name, instead of multiply the metrics by added suffixes `_total`, `_sum`, `_max` etc.
  
- `receiver/ciscoosreceiver`: `ciscoosreceiver`: Add new receiver for collecting metrics from Cisco network devices via SSH (#42647)
  Supports Cisco IOS, IOS-XE, and NX-OS devices with SSH-based metric collection.
  Initial implementation includes system scraper for device availability and connection metrics.
  
- `receiver/gitlab`: Promote GitLab receiver to Alpha stability (#41592)
- `receiver/jmx`: Add JMX metrics gatherer version 1.51.0-alpha (#43666)
- `receiver/jmx`: Add JMX scraper version 1.51.0-alpha (#43667)
- `receiver/pprof`: convert google/pprof to OTel profiles (#42843)
- `receiver/redfish`: this branch provides the first concrete implementation of the new component (#33724)

### 🧰 Bug fixes 🧰

- `exporter/clickhouse`: Fix a bug in the exporter factory resulting in a nil dereference panic when the clickhouse.json feature gate is enabled (#43733)
- `exporter/kafka`: franz-go: Fix underreported kafka_exporter_write_latency metric (#43803)
- `exporter/loadbalancing`: Fix high cardinality issue in loadbalancing exporter by moving endpoint from exporter ID to attributes (#43719)
  Previously, the exporter created unique IDs for each backend endpoint by appending the endpoint
  to the exporter ID (e.g., loadbalancing_10.11.68.62:4317). This caused high cardinality in metrics,
  especially in dynamic environments. Now the endpoint is added as an attribute instead.
  
- `exporter/pulsar`: Fix the oauth2 flow for pulsar exporter by adding additional configuration fields (#435960)
  Fixes the oauth2 authentication flow in pulsar exporter by exposing additional configuration like `private_key` and `scope`.
- `processor/metricstarttime`: Do not set start timestamp if it is already set. (#43739)
- `processor/tail_sampling`: Fix panic when invalid regex was sent to string_attribute sampler (#43735)
- `receiver/awss3`: Fix S3 prefix trimming logic in awss3reader to correctly handle empty, single slash '/', and double slash '//' prefixes. (#43587)
  This fix ensures the S3 object prefix is generated consistently for all prefix formats (e.g., `""`, `/`, `//`, `/logs/`, `//raw//`),
  preventing malformed S3 paths when reading from buckets with non-standard prefixes.
  
- `receiver/hostmetrics`: Allow process metrics to be recorded if the host does not have cgroup functionality (#43640)
- `receiver/kafka`: Corrected the documentation for the Kafka receiver to accurately the supported/default group balancer strategies. (#43892)
- `receiver/postgresql`: Change the unit of the metric `postgresql.table.vacuum.count` to be `vacuum` instead of vacuums (#43272)
- `receiver/prometheus`: Fix missing staleness tracking leading to missing no recorded value data points. (#43893)
- `receiver/prometheusremotewrite`: Fixed a concurrency bug in the Prometheus remote write receiver where concurrent requests with identical job/instance labels would return empty responses after the first successful request. (#42159)
- `receiver/pulsar`: Fix the oauth2 flow for pulsar exporter by adding additional configuration fields (#43596)
  Fixes the oauth2 authentication flow in pulsar receiver by exposing additional configuration like `private_key` and `scope`.
  
- `receiver/receiver_creator`: Fix annotation-discovery config unmarshaling for nested configs (#43730)

<!-- previous-version -->


## v1.45.0/v0.139.0

### 🛑 Breaking changes 🛑

- `cmd/mdatagen`: Make stability.level a required field for metrics (#14070)
- `cmd/mdatagen`: Replace `optional` field with `requirement_level` field for attributes in metadata schema (#13913)
  The `optional` boolean field for attributes has been replaced with a `requirement_level` field that accepts enum values: `required`, `conditionally_required`, `recommended`, or `opt_in`.
  - `required`: attribute is always included and cannot be excluded
  - `conditionally_required`: attribute is included by default when certain conditions are met (replaces `optional: true`)
  - `recommended`: attribute is included by default but can be disabled via configuration (replaces `optional: false`)
  - `opt_in`: attribute is not included unless explicitly enabled in user config
  When `requirement_level` is not specified, it defaults to `recommended`.
  
- `pdata/pprofile`: Remove deprecated `PutAttribute` helper method (#14082)
- `pdata/pprofile`: Remove deprecated `PutLocation` helper method (#14082)

### 💡 Enhancements 💡

- `all`: Add FIPS and non-FIPS implementations for allowed TLS curves (#13990)
- `cmd/builder`: Set CGO_ENABLED=0 by default, add the `cgo_enabled` configuration to enable it. (#10028)
- `pkg/config/configgrpc`: Errors of type status.Status returned from an Authenticator extension are being propagated as is to the upstream client. (#14005)
- `pkg/config/configoptional`: Adds new `configoptional.AddEnabledField` feature gate that allows users to explicitly disable a `configoptional.Optional` through a new `enabled` field. (#14021)
- `pkg/exporterhelper`: Replace usage of gogo proto for persistent queue metadata (#14079)
- `pkg/pdata`: Remove usage of gogo proto and generate the structs with pdatagen (#14078)

### 🧰 Bug fixes 🧰

- `exporter/debug`: add queue configuration (#14101)

<!-- previous-version -->
