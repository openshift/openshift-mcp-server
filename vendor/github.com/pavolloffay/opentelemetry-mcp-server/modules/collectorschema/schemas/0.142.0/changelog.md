## v0.142.0

### 🛑 Breaking changes 🛑

- `all`: It's recommended to change the field type in your component configuration to be `configoptional.Optional[exporterhelper.QueueBatchConfig]` to keep the `enabled` subfield. Use configoptional.Some(exporterhelper.NewDefaultQueueConfig()) to enable by default. Use configoptional.Default(exporterhelper.NewDefaultQueueConfig()) to disable by default. (#44320)
- `exporter/datadog`: Update the Datadog exporter to support the Orchestrator Explorer by accepting receiver/k8sobjects logs and sending Kubernetes data to the Datadog endpoint. (#44523)
  The Cluster name field in Orchestrator Explorer Configuration has been removed. Use the k8s.cluster.name attribute instead.
- `exporter/prometheusremotewrite`: Updated to Remote Write 2.0 spec rc.4, requiring Prometheus 3.8.0 or later as the receiving endpoint. (#44861)
  The upstream Prometheus library updated the Remote Write 2.0 protocol from rc.3 to rc.4 in
  prometheus/prometheus#17411. This renamed `CreatedTimestamp` to `StartTimestamp` and moved it
  from the `TimeSeries` message to individual `Sample` and `Histogram` messages. This is a
  wire-protocol incompatibility, so if you are exporting to a Prometheus server, you must upgrade
  it to version 3.8.0 or later to receive data from this exporter.
  
- `extension/googlecloudlogentry_encoding`: Parse External Application Load Balancer logs into log record attributes instead of placing it in the body as is. (#44438)
- `pkg/stanza`: Allow `max_batch_size` of 0 for unlimited batching in `recombine` operator (#43982)
  The recombine operator now supports setting `max_batch_size: 0` to disable batch size limits.
  This allows unlimited batching, letting entries be combined based only on `max_log_size` and matching conditions.
  If you have `max_batch_size: 0` in your config and want to keep the behavior unchanged, change the configuration to `max_batch_size: 1`.
  
- `processor/cumulativetodelta`: Change default `max_staleness` from 0 (infinite) to 1 hour (#44427)
  The processor now defaults to a `max_staleness` of 1 hour instead of 0 (infinite retention).
  This prevents unbounded memory growth in long-running collector instances, especially when tracking metrics with high cardinality or frequently changing attribute values.
  To restore the previous behavior of infinite retention, explicitly set `max_staleness: 0` in your configuration.
  
- `processor/resourcedetection`: Promote `processor.resourcedetection.propagateerrors` feature gate to beta (#44609)
- `processor/resourcedetection`: Remove deprecated `attributes` configuration option (#44610)
- `receiver/awss3`: Remove the `s3_partition` config option in favor of `s3_partition_format` and `s3_partition_timezone` options. This aligns the S3 receiver more closely with the S3 Exporter. Also add the ability to include or exclude the telemetry type from the file prefix using the `file_prefix_include_telemetry_type` option. (#43720)
- `receiver/docker_stats`: Upgrades default Docker API version to 1.44 to be compatible with recent Docker Engine versions. (#44279)
  Users requiring an older Docker API version can set the `api_version` in the docker stats receiver config. The minimum supported API level is not changed, only default.
- `receiver/filelog`: Move `filelog.decompressFingerprint` to stable stage (#44570)
- `receiver/prometheus`: Promote the receiver.prometheusreceiver.RemoveStartTimeAdjustment feature gate to stable and remove in-receiver metric start time adjustment in favor of the metricstarttime processor, including disabling the created-metric feature gate. (#44180)
  Previously, users could disable the RemoveStartTimeAdjustment feature gate to temporarily keep the legacy start time adjustment behavior in the Prometheus receiver.
  With this promotion to stable and bounded registration, that gate can no longer be disabled; the receiver will no longer set StartTime on metrics based on process_start_time_seconds, and users should migrate to the metricstarttime processor for equivalent functionality.
  This change also disables the receiver.prometheusreceiver.UseCreatedMetric feature gate, which previously used the `<metric>_created` series to derive start timestamps for counters, summaries, and histograms when scraping non OpenMetrics protocols.
  However, this does not mean that the `_created` series is always ignored: when using the OpenMetrics 1.0 protocol, Prometheus itself continues to interpret the `_created` series as the start timestamp, so only the receiver-side handling for other scrape protocols has been removed.
  
- `receiver/prometheus`: Native histogram scraping and ingestion is now controlled by the scrape configuration option `scrape_native_histograms`. (#44861)
  The feature gate `receiver.prometheusreceiver.EnableNativeHistograms` is now stable and enabled by default.
  Native histograms scraped from Prometheus will automatically be converted to OpenTelemetry exponential histograms.
  
  To enable scraping of native histograms, you must configure `scrape_native_histograms: true` in your Prometheus
  scrape configuration (either globally or per-job). Additionally, the protobuf scrape protocol must be enabled
  by setting `scrape_protocols` to include `PrometheusProto`.
  
- `receiver/prometheusremotewrite`: Updated to Remote Write 2.0 spec rc.4, requiring Prometheus 3.8.0 or later (#44861)
  The upstream Prometheus library updated the Remote Write 2.0 protocol from rc.3 to rc.4 in
  prometheus/prometheus#17411. This renamed `CreatedTimestamp` to `StartTimestamp` and moved it
  from the `TimeSeries` message to individual `Sample` and `Histogram` messages. This is a
  wire-protocol incompatibility, so Prometheus versions 3.7.x and earlier will no longer work
  correctly with this receiver. Please upgrade to Prometheus 3.8.0 or later.
  

### 🚩 Deprecations 🚩

- `processor/k8sattributes`: Removes stable k8sattr.fieldExtractConfigRegex.disallow feature gate (#44694)
- `receiver/kafka`: Deprecate `default_fetch_size` parameter for franz-go client (#43104)
  The `default_fetch_size` parameter is now deprecated for the franz-go Kafka client and will only be used with the legacy Sarama client.
  Users should configure `max_fetch_size` instead when using franz-go.
  This deprecation is marked as of v0.142.0.
  
- `receiver/kafka`: Support configuring a list of topics and exclude_topics; deprecate topic and exclude_topic (#44477)
- `receiver/prometheus`: Deprecate `use_start_time_metric` and `start_time_metric_regex` config in favor of the processor `metricstarttime` (#44180)

### 🚀 New components 🚀

- `receiver/yanggrpc`: Implement the YANG/gRPC receiver (#43840)

### 💡 Enhancements 💡

- `exporter/elasticsearch`: add dynamic data stream routing for connectors (#44525)
- `exporter/kafka`: Adds server.address attribute to all Kafka exporter metrics. (#44649)
- `exporter/prometheusremotewrite`: Add option to remove `service.name`, `service.instance.id`, `service.namespace` ResourceAttribute from exported metrics (#44567)
- `exporter/signalfx`: Support setting default properties for dimension updates to be set lazily as part of configuration (#44891)
- `extension/azure_encoding`: Implement general Azure Resource Log parsing functionality (#41725)
- `extension/datadog`: Datadog Extension users may view and manage OTel Collectors in Fleet Automation. (#44666)
  Interested users should read [the post on the Datadog Monitor blog](https://www.datadoghq.com/blog/manage-opentelemetry-collectors-with-datadog-fleet-automation/) and fill out the preview intake form listed there.
  
- `extension/datadog`: Adds deployment_type configuration option to the Datadog Extension. (#44430)
  Users may specify the deployment type of the collector in Datadog Extension configuration to view in Datadog app.
  If the collector is deployed as a gateway (i.e. receiving pipeline telemetry from multiple hosts/sources),
  user should specify "gateway" as the deployment type.
  If the collector is deployed as a daemonset/agent, user should specify "daemonset" as the deployment type.
  The default setting is "unknown" if not set.
  
- `extension/datadog`: Adds standard (non-billed) liveness metric `otel.datadog_extension.running` to ensure host data is shown in Datadog app. (#44285)
- `extension/googlecloudlogentry_encoding`: Add support for GCP VPC Flow Log fields for MIG (Managed Instance Group) and Google Service logs. (#44220)
  Adds support for the following GCP VPC Flow Log fields:
  - Add support for gcp.vpc.flow.{source,destination}.google_service.{type,name,connectivity} 
  - Add support for gcp.vpc.flow.{source,destination}.instance.managed_instance_group.{name,region,zone}
  
- `extension/health_check`: Added extension.healthcheck.useComponentStatus feature gate to enable v2 component status reporting in healthcheckextension while maintaining backward compatibility by default. (#42256)
- `pkg/ottl`: Accept string trace/span/profile IDs for `TraceID()`, `SpanID()`, and `ProfileID()` in OTTL. (#43429)
  This change allows for a more straightforward use of string values to set trace, span, and profile IDs in OTTL.
- `pkg/stanza`: New featuregate `filelog.windows.caseInsensitive` introduced. It will make glob matching is case-insensitive on Windows. (#40685)
  Previously, any `include` pattern that included some manner of wildcard (`*` or `**`) would
  be case-sensitive on Windows, but Windows filepaths are by default case-insensitive. This meant
  that in a directory with the files `a.log` and `b.LOG`, the pattern `*.log` would previously only
  match `a.log`. With the `filelog.windows.caseInsensitive` featuregate enabled, it will match both `a.log`
  and `b.LOG` when on Windows. The behaviour is the same as always on other operating systems, as all other
  currently supported platforms for the Collector have case-sensitive filesystems.
  
- `pkg/translator/azurelogs`: Added support for Activity Logs Recommendation category (#43220)
- `processor/k8sattributes`: Updates semconv version to v1.37.0 (#44696)
- `processor/resourcedetection`: Add support for dynamic refresh resource attributes with refresh_interval parameter (#42663)
- `processor/resourcedetection`: Update semconv dependency to 1.37.0 which updates the schema url in the data, but no other impact is expected. (#44726)
- `processor/transform`: New Transform Processor function `set_semconv_span_name()` to overwrite the span name with the semantic conventions for HTTP, RPC, messaging, and database spans. (#43124)
  In other cases, the original `span.name` is unchanged.
  The primary use of `set_semconv_span_name()` is alongside the
  [Span Metrics Connector](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/spanmetricsconnector)
  to address high-cardinality issues in span metrics when `span.name` does not comply with the OTel requirement
  that span names be low cardinality.
  
- `receiver/azuremonitor`: Scrape storage account sub types (#37220)
- `receiver/datadog`: Improve the compliance with OTel Semantic Conventions for RPC spans in the Datadog receiver. (#44671)
  Compliance improvements on spans received via the Datadog receiver when applicable:
  * Set span name according to RPC conventions.
  * Set `rpc.method` and `rpc.service` attributes.
  
- `receiver/datadog`: Bump OTel Semantic Conventions from v1.30.0 to v1.37.0 in the Datadog receiver. (#44671)
  Changes in OTel Semantic Conventions v1.37.0 affecting the Datadog receiver:
  * The Datadog tag `runtime` is now mapped to the OTel attribute `container.runtime.name` instead of `container.runtime`.
  
- `receiver/datadog`: Improve the compliance with OTel Semantic Conventions for AWS SDK spans in the Datadog receiver. (#44723)
  Compliance improvements on spans received via the Datadog receiver when applicable:
  * Set span name according to AWS SDK conventions.
  * Set `rpc.system`, `rpc.method` and `rpc.service` attributes.
  
- `receiver/datadog`: Add `receiver.datadogreceiver.EnableMultiTagParsing` feature gate (#44747)
  The feature flag changes the logic that converts Datadog tags to OpenTelemetry attributes.
  When the flag is enabled, data points that have multiple tags starting with the same `key:` prefix
  will be turned into an attribute slice (instead of a string) containing all the suffix values.
  
- `receiver/datadog`: Improve the compliance with OTel Semantic Conventions for HTTP spans in the Datadog receiver. (#44722)
  Compliance improvements on spans received via the Datadog receiver when applicable:
  Set span name according to HTTP conventions for `web.request` and `http.request` spans.
  
- `receiver/github`: Add concurrency limiting to reduce likelihood of hitting secondary rate limits (#43388)
  Adds `concurrency_limit` configuration parameter (default: 50) to limit
  concurrent repository processing goroutines. This reduces the likelihood of
  getting 502/504 errors when scraping organizations with >100 repositories.
  
- `receiver/googlecloudpubsub`: Exponential backoff streaming restarts (#44741)
- `receiver/kafka`: Make `session_timeout`, `heartbeat_interval`, `max_partition_fetch_size`, and `max_fetch_wait` unconditional in franz-go consumer (#44839)
- `receiver/kafka`: Validate that `exclude_topics` entries in kafkareceiver config are non-empty. (#44920)
- `receiver/oracledb`: Added independent collection interval config for Oracle top query metrics collection (#44607)
- `receiver/prometheusremotewrite`: Map.PutStr causes excessive memory allocations due to repeated slice expansions (#44612)
- `receiver/splunk_hec`: Support parsing JSON array payloads in Splunk HEC receiver (#43941)
- `receiver/sshcheck`: Promote sshcheck receiver to beta stability (#41573)
- `receiver/yanggrpc`: Promote to alpha stability (#44783)

### 🧰 Bug fixes 🧰

- `exporter/elasticsearch`: Fix hostname mapping in Elasticsearch exporter (#44874)
  - The exporter now supports to map an otel field to an ecs field only if the ecs field is not already present. This is applied to `host.hostname` mapping. 
  
- `processor/cumulativetodelta`: Check whether bucket bounds are the same when verifying whether histograms are comparable (#44793)
- `processor/cumulativetodelta`: Fix logic handling ZeroThreshold increases for exponential histograms (#44793)
- `processor/filter`: Fix context initialization for metric/datapoint context (#44813)
- `processor/k8sattributes`: Fix `k8sattr.labelsAnnotationsSingular.allow` feature gate to affect config default tag names in addition to runtime extraction (#39774)
- `processor/tail_sampling`: Fix a memory leak introduced in 0.141.0 of the tail sampling processor when not blocking on overflow. (#44884)
- `receiver/datadog`: The `db.instance` tag of Datadog database client spans should be mapped to the OTel attribute `db.namespace`, not to `db.collection.name`. (#44702)
  Compliance improvements on spans received via the Datadog receiver when applicable:
  * The `db.instance` tag is now mapped to the OTel attribute `db.namespace` instead of `db.collection.name`.
  * The `db.sql.table` tag is mapped to the OTel attribute `db.collection.name`.
  * The `db.statement` tag is mapped to the OTel attribute `db.query.text`.
  
- `receiver/datadog`: Fix Datadog trace span counting so otelcol_receiver_accepted_spans is not under-reported (#44865)
  Previously only the last payload's spans were counted, so the otelcol_receiver_accepted_spans metric could be lower than otelcol_exporter_sent_spans in pipelines where they should match.
  
- `receiver/github`: Adds corrections to span kind for GitHub events when they are tasks. (#44667)
- `receiver/googlecloudpubsub`: Acknowledge messages at restart (#44706)
  Rewrote the control flow loop so the acknowledgment of messages is more reliable. At stream restart, the messages
  ackIds are resent immediately without an explicit acknowledgment. Outstanding ackIds are only cleared when the
  acknowledgment is sent successfully.
  
- `receiver/googlecloudspanner`: Fixed goroutine leaks in ttlcache lifecycle management and applied modernize linter fixes across multiple receivers. (#44779)
  - Simplified cache lifecycle management by removing unnecessary WaitGroup complexity
  - Added goleak ignores for ttlcache goroutines that don't stop immediately after Stop()
  
- `receiver/kafka`: Use `max_fetch_size` instead of `default_fetch_size` in franz-go client (#43104)
  The franz-go Kafka consumer was incorrectly using `default_fetch_size` (a Sarama-specific setting) instead of `max_fetch_size` when configuring `kgo.FetchMaxBytes`.
  This fix ensures the correct parameter is used and adds validation to prevent `max_fetch_size` from being less than `min_fetch_size`.
  The default value for `max_fetch_size` has been changed from 0 (unlimited) to 1048576 (1 MiB) to maintain backward compatibility with the previous (incorrect) behavior.
  
- `receiver/prometheus`: Fix HTTP response body leak in target allocator when fetching scrape configs fails (#44921)
  The getScrapeConfigsResponse function did not close resp.Body on error paths.
  If io.ReadAll or yaml.Unmarshal failed, the response body would leak,
  potentially causing HTTP connection exhaustion.
  
- `receiver/prometheus`: Fixes yaml marshaling of prometheus/common/config.Secret types (#44445)

<!-- previous-version -->


## v1.48.0/v0.142.0

### 💡 Enhancements 💡

- `exporter/debug`: Add logging of dropped attributes, events, and links counts in detailed verbosity (#14202)
- `extension/memory_limiter`: The memorylimiter extension can be used as an HTTP/GRPC middleware. (#14081)
- `pkg/config/configgrpc`: Statically validate gRPC endpoint (#10451)
  This validation was already done in the OTLP exporter. It will now be applied to any gRPC client.
  
- `pkg/service`: Add support to disabling adding resource attributes as zap fields in internal logging (#13869)
  Note that this does not affect logs exported through OTLP.
  

<!-- previous-version -->
