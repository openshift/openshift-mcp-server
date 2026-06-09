## v0.135.0

### 🛑 Breaking changes 🛑

- `apachereceiver`: Add number of connections per async state metrics. (#41886)
- `githubreceiver`: Update semantic conventions from v1.27.0 to v1.37.0 with standardized VCS and CICD attributes (#42378)
  - Resource attributes changed: `organization.name` -> `vcs.owner.name`, `vcs.vendor.name` -> `vcs.provider.name`
  - Trace attributes now use standardized VCS naming: `vcs.ref.head.type` -> `vcs.ref.type`
  - Added new standardized attributes: `vcs.ref.base.name`, `vcs.ref.base.type`, `vcs.ref.type`
  - Delta metrics now include base reference tracking attributes
  - Updated schema URL to https://opentelemetry.io/schemas/1.37.0
  
- `k8sattributesprocessor`: Introduce allowLabelsAnnotationsSingular feature gate to use singular format for k8s label and annotation resource attributes (#39774)
  The feature gate, when enabled, will change the default resource attribute key format from k8s.<workload>.labels.<label-key> to k8s.<workload>.label.<label-key>. Same applies for annotations.
- `receiver/sapm`: The `SAPM Receiver` component has been removed from the repo and is no longer being published as it has been deprecated since 22nd October 2024 and the removal date of April 2025 has passed. (#41411)

### 💡 Enhancements 💡

- `transformprocessor`: Add support for merging histogram buckets. (#40280)
  The transformprocessor now supports merging histogram buckets using the `merge_histogram_buckets` function.
  
- `k8seventsreceiver`: Adds scope name and version to logs (#42426)
- `googlecloudlogentry_encoding`: Add support for request attributes and destination attributes in cloud audit logs (#42160)
- `azureeventhubreceiver`: Added feature flag to use the new Azure SDK (#40795)
- `dockerstatsreceiver`: Add Windows support (#42297)
  The dockerstatsreceiver now supports Windows hosts.
  
- `elasticsearchexporter`: Populate profiling-hosts index with resource attribute information. (#42220)
- `tinybirdexporter`: Limit request body to 10MB to avoid exceeding the EventsAPI size limit. (#41782)
- `exporter/kafkaexporter`: Use franz-go client for Kafka exporter as default, promoting the exporter.kafkaexporter.UseFranzGo feature gate to Beta. (#42156)
- `exporter/kafka`: Add allow_auto_topic_creation producer option to kafka exporter and client (#42468)
- `processor/resourcedetection`: Add support for hetzner cloud in resourcedetectionprocessor (#42476)
- `kafkareceiver`: Add `rack_id` configuration option to enable rack-aware replica selection (#42313)
  When configured and brokers support rack-aware replica selection, the client will prefer fetching from the closest replica, potentially reducing latency and improving performance.
  
- `statsdreceiver`: Introduce explicit bucket for statsd receiver (#41203, #41503)
- `coreinternal/aggregateutil`: Aggregate exponential histogram data points when different offsets are present (#42412)
- `prometheusremotewriteexporter`: Remove unnecessary buffer copy in proto conversion (#42329)
- `pkg/translator/prometheusremotewrite`: `FromMetricsV2` now supports translating exponential histograms. (#33661)
  The translation layer for Prometheus remote write 2 now supports exponential histograms but is not fully implemented and ready for use.
- `processor/k8sattributes`: Support extracting labels and annotations from k8s DaemonSets (#37957)
- `processor/k8sattributes`: Support extracting labels and annotations from k8s Jobs (#37957)
- `k8sclusterreceiver`: Add option `namespaces` for setting a list of namespaces to be observed by the receiver. This supersedes the `namespace` option which is now deprecated. (#40089)
- `k8sobjectsreceiver`: Adds the instrumentation scope name and version (#42290)
- `receiver/kubeletstats`: Introduce k8s.pod.volume.usage metric. (#40476)
- `datadogexporter`: Add alpha feature gate 'exporter.datadogexporter.InferIntervalForDeltaMetrics'. (#42494)
  This feature gate will set the interval for OTLP delta metrics mapped by the exporter when it can infer them.
  
- `sqlserverreceiver`: Add `service.instance.id` resource attribute to all metrics and logs (#41894)
  The `service.instance.id` attribute is added in the format `<host>:<port>` to uniquely identify 
  SQL Server hosts.
  

### 🧰 Bug fixes 🧰

- `awslogsencodingextension`: Fixed gzip header detection for mixed compressed/uncompressed files (#41884)
  The extension now properly detects gzip magic bytes (0x1f, 0x8b) before attempting decompression,
  preventing "gzip: invalid header" errors when processing files with .gz extensions that are not actually compressed.
  Affected formats: WAF logs, CloudTrail logs, CloudWatch subscription filter logs, and VPC Flow logs.
  
- `opampsupervisor`: Always respond to `RemoteConfig` messages with a `RemoteConfigStatus` message (#42474)
  Previously the Supervisor would not respond if the effective config did not change.
  This caused issues where the same config with a different hash (e.g. reordered keys in the config)
  would not be reported and would appear unapplied by the Supervisor.
  
- `elasticsearchexporter`: Ignore expected errors when making bulk requests to profiling indices. (#38598)
- `libhoneyreceiver`: Properly handle compressed payloads (#42279)
  Compression issues now return a 400 status rather than panic. Exposes the http library's compression algorthms to let users override if needed.
- `libhoneyreceiver`: Allow service.name with unset scope.name (#42432)
  This change allows the receiver to handle multiple service.names even if there are spans without the scope set. It also avoids a panic when a downstream consumer is missing.

<!-- previous-version -->


## v1.41.0/v0.135.0

### 💡 Enhancements 💡

- `exporterhelper`: Add new `exporter_queue_batch_send_size` and `exporter_queue_batch_send_size_bytes` metrics, showing the size of telemetry batches from the exporter. (#12894)

<!-- previous-version -->
