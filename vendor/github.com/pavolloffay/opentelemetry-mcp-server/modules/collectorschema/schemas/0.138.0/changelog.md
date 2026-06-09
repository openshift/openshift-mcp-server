## v0.138.0

### 🛑 Breaking changes 🛑

- `connector/datadog`: Mark NativeIngest as stable. (#104622)
- `connector/signaltometrics`: Drop `signaltometrics.service.{name, namespace}` resource attribute from produced metrics. (#43148)
- `exporter/datadog`: Remove `logs::dump_payloads` config option from `datadogexporter` config. (#43427)
  Please remove the previously deprecated `logs::dump_payloads` config option from your `datadogexporter` config.
- `exporter/elasticsearch`: Remove batcher and related config in favor of sending queue (#42718)
  Previously deprecated `batcher` configuration is removed. `num_consumers` and `flush` are now deprecated as they conflict with `sending_queue` configurations.
- `extension/googlecloudlogentry_encoding`: Parse VPC flow logs into log record attributes instead of placing them in the body directly. (#43017)

### 🚀 New components 🚀

- `receiver/icmpcheckreceiver`: Add initial skeleton of ICMP check receiver (README, config, factory, metadata) with In Development stability. (#29009)
- `receiver/redfish`: adds a redfish receiver (#33724)

### 💡 Enhancements 💡

- `all`: Changelog entries will now have their component field checked against a list of valid components. (#43179)
  This will ensure a more standardized changelog format which makes it easier to parse.
- `cmd/telemetrygen`: Enable creation of attributes with values of slice type (#39018)
- `exporter/coralogix`: Add HTTP/protobuf protocol support alongside existing gRPC transport. (#43216)
  The exporter now supports both gRPC (default) and HTTP/protobuf protocols for sending telemetry data.
  HTTP transport enables proxy support and provides an alternative for environments where gRPC is restricted.
  Configure using the `protocol` field with values "grpc" or "http".
  
- `exporter/datadog`: Make defaults for `sending_queue::batch` section to work out of the box with Datadog API intake limits. (#43082)
- `exporter/elasticsearch`: Support experimental 'encoding.format' scope attribute for dataset routing. (#42844)
- `exporter/kafka`: Add support for partitioning log records by trace ID (#39146)
- `exporter/prometheus`: Enable `sending_queue` section for the Prometheus exporter. (#42629)
- `extension/awslogs_encoding`: Add feature gate to set aws.vpc.flow.start timestamp field to ISO8601 format (#43392)
  Feature gate ID: extension.awslogsencoding.vpcflow.start.iso8601
  When enabled, the aws.vpc.flow.start field will be formatted as an ISO-8601 string 
  instead of a Unix timestamp integer in seconds since epoch. Default behavior remains unchanged for backward compatibility.
  Enable with: --feature-gates=extension.awslogsencoding.vpcflow.start.iso8601
  
- `extension/encoding`: Add user_agent.original, destination.address, destination.port, url.domain to ELB access logs (#43141)
- `internal/kafka`: Log a hint when broker connections fail due to possible TLS misconfiguration (#40145)
- `pkg/ottl`: Add XXH3 Converter function to converts a `value` to a XXH3 hash/digest (#42792)
- `pkg/sampling`: Note that pkg/sampling implements the new OpenTelemetry specification (#43396)
- `processor/filter`: Add profiles support (#42762)
- `processor/isolationforest`: Add adaptive window sizing feature that automatically adjusts window size based on traffic patterns, memory usage, and model stability (#42751)
  The adaptive window sizing feature enables dynamic adjustment of the isolation forest sliding window size based on:
  - Traffic velocity and throughput patterns
  - Memory usage and resource constraints  
  - Model stability and performance metrics
  This enhancement improves resource utilization and anomaly detection accuracy for varying workload patterns.
  
- `processor/resourcedetection`: Add Openstack Nova resource detector to gather Openstack instance metadata as resource attributes (#39117)
  The Openstack Nova resource detector has been added to gather metadata such as host name, ID, cloud provider, region, and availability zone as resource attributes, enhancing the observability of Openstack environments.
- `processor/resourcedetection`: Add Azure availability zone to resourcedetectionprocessor (#40983)
- `receiver/azuremonitor`: parallelize calls by subscriptions in Batch API mode (#39417)
- `receiver/ciscoosreceiver`: Add `ciscoosreceiver` to collect metrics from Cisco OS devices via SSH (#42647)
  Supports SSH-based metric collection from Cisco devices including:
  - System metrics (CPU, memory utilization)
  - Interface metrics (bytes, packets, errors, status)
  - Configurable scrapers for modular metric collection
  - Device authentication via password or SSH key
  
- `receiver/gitlab`: Add span attributes in GitLab receiver (#35207)
- `receiver/hostmetrics`: Add metrics, Linux scraper, and tests to hostmetricsreceiver's nfsscraper (#40134)
- `receiver/icmpcheckreceiver`: Add complete scraping implementation with ICMP ping/echo to collect metrics (#29009)
  Replaces skeleton implementation with full production-ready collector functionality.
  Includes metrics metadata and completed configuration.
  Includes real scraper implementation that performs ICMP checks and collects metrics.
  Includes README docs.
  
- `receiver/mysql`: Support query-level collection. (#41847)
  Added top query (most time consumed) collection. The query will gather the queries took most of the time during the last
  query interval and report related metrics. The number of queries can be configured. This will enable user to have better
  understanding on what is going on with the database. This enhancement empowers users to not only monitor but also actively 
  manage and optimize their MySQL database performance based on real usage patterns.
  
- `receiver/prometheus`: added NHCB(native histogram wit custom buckets) to explicit histogram conversion (#41131)
- `receiver/redis`: Add `ClusterInfo` capability to `redisreceiver` (#38117)
- `receiver/splunkenterprise`: Added a new metric `splunk.license.expiration.seconds_remaining` to report the time remaining in seconds before a Splunk Enterprise license expires. (#42630)
  - Includes the following attributes: `status`, `label`, `type`.
  
- `receiver/sqlserver`: Removing instance name usage in the SQL for top-query collection. (#43558)
  Additional config of instance name is not required for collecting the top queries.
  
- `receiver/syslog`: Promote Syslog receiver to beta stability (#28551)

### 🧰 Bug fixes 🧰

- `exporter/awss3`: Support compression with the sumo_ic marshaller (#43574)
- `exporter/elasticsearch`: Ensure metadata keys are always propagated in client context with batching enabled. (#41937)
- `exporter/prometheus`: Fixes data_type field formatting in the error logs message when exporting  unknown metrics types - e.g. native histograms. (#43595)
- `exporter/syslog`: Fix timestamp formatting in rfc5424 syslog messages to use microsecond precision (#43114)
- `processor/metricstarttime`: Fixes bug where adjustment only relied on the DoubleValue and ignored the IntValue (#42202)
- `receiver/k8s_cluster`: Fix for k8sclusterreceiver to handle empty containerID in ContainerStatus (#43147)
- `receiver/libhoney`: fix panic when decompressing poorly formatted data (#42272)
  When decompressing poorly formatted data, the receiver would panic. This has now been fixed.
- `receiver/oracledb`: Fix to use time from database clock for more accurate collection window calculation. (#43621)
  Fixed the top-query collection logic to use database clock instead of the time from collector instance.
  

<!-- previous-version -->


## v1.44.0/v0.138.0

### 🛑 Breaking changes 🛑

- `all`: Remove deprecated type `TracesConfig` (#14036)
- `pkg/exporterhelper`: Add default values for `sending_queue::batch` configuration. (#13766)
  Setting `sending_queue::batch` to an empty value now results in the same setup as the default batch processor configuration.
  
- `all`: Add unified print-config command with mode support (redacted, unredacted), json support (unstable), and validation support. (#11775)
  This replaces the `print-initial-config` command. See the `service` package README for more details. The original command name `print-initial-config` remains an alias, to be retired with the feature flag.

### 💡 Enhancements 💡

- `all`: Add `keep_alives_enabled` option to ServerConfig to control HTTP keep-alives for all components that create an HTTP server. (#13783)
- `pkg/otelcol`: Avoid unnecessary mutex in collector logs, replace by atomic pointer (#14008)
- `cmd/mdatagen`: Add lint/ordering validation for metadata.yaml (#13781)
- `pdata/xpdata`: Refactor JSON marshaling and unmarshaling to use `pcommon.Value` instead of `AnyValue`. (#13837)
- `pkg/exporterhelper`: Expose `MergeCtx` in exporterhelper's queue batch settings` (#13742)

### 🧰 Bug fixes 🧰

- `all`: Fix zstd decoder data corruption due to decoder pooling for all components that create an HTTP server. (#13954)
- `pkg/otelcol`: Remove UB when taking internal logs and move them to the final zapcore.Core (#14009)
  This can happen because of a race on accessing `logsTaken`.
- `pkg/confmap`: Fix a potential race condition in confmap by closing the providers first. (#14018)

<!-- previous-version -->
