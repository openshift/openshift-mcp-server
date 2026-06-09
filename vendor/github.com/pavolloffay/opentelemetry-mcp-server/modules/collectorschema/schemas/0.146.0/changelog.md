## v0.146.0

### 🛑 Breaking changes 🛑

- `all`: Increase minimum Go version to 1.25 (#46000)
- `exporter/elasticsearch`: Remove ecs mode log enrichment for `agent.name` and `agent.version` (#45849)
  The removed log enrichments are duplicate code that already exists in the `github.com/elastic/opentelemetry-collector-components/processor/elasticapmprocessor`. It is recommended to use the `github.com/elastic/opentelemetry-collector-components/processor/elasticapmprocessor` when using mapping mode `ecs` to ensure index documents contain all required Elastic fields to power the Kibana UI.
- `processor/akamaidetector`: Update semantic conventions to v1.39.0 and use the convention for cloud.platform in Akamai detector (#45511)
- `processor/k8s_attributes`: Introduce semantic conventions compliant feature gate pair for k8sattributes processor (#44693)
  - Added `processor.k8sattributes.EmitV1K8sConventions` feature gate to enable stable semantic convention attributes (singular form: `k8s.<workload>.label.<key>` and `k8s.<workload>.annotation.<key>`)
  - Added `processor.k8sattributes.DontEmitV0K8sConventions` feature gate to disable legacy non-compliant attributes (plural form: `k8s.<workload>.labels.<key>` and `k8s.<workload>.annotations.<key>`)
  - Both feature gates are in `alpha` stage and disabled by default
  - The processor now validates that legacy attributes cannot be disabled without enabling stable attributes
  - Deprecated `k8sattr.labelsAnnotationsSingular.allow` feature gate in favor of the new semconv-compliant gates (will be removed in v0.150.0)
  - During migration period, both legacy and stable attributes can coexist when `EmitV1K8sConventions` is enabled but `DontEmitV0K8sConventions` is not
  
- `processor/resourcedetection`: Promote `processor.resourcedetection.propagateerrors` feature gate to Stable and is now always enabled (#44609)
- `receiver/hostmetrics`: `process.context_switches` will now properly count context switches for all threads. (#36804)
  Previously, only the lead thread's context switches would be counter. We believe this was a bug,
  but are marking it as a breaking change since the values of these metrics may change drastically
  compared to previous expectations. However, the values you get now will be more accurate than before.
- `receiver/kafkametrics`: `receiver.kafkametricsreceiver.UseFranzGo` feature gate is now in Beta stage and enabled by default (#44600)

### 🚩 Deprecations 🚩

- `connector/signal_to_metrics`: Rename `signaltometrics` connector to `signal_to_metrics` and add deprecated alias `signal_to_metrics` (#45551)
- `exporter/alibabacloud_logservice`: Marking component unmaintained (#46029)
- `processor/datadogsemantics`: Deprecate the `datadogsemantics` processor. (#46052)
  If you are using this component, please contact [Datadog support](https://www.datadoghq.com/support/).
  
- `processor/k8s_attributes`: Rename `k8sattributes` processor to `k8s_attributes` processor and add deprecated alias `k8sattributes`. (#45894)
- `receiver/kubeletstats`: The `GCEPersistentDisk`, `AWSElasticBlockStore`, and `Glusterfs` have been deprecated as these have been depreacted in k8s (#40477)
  K8s documentation for the three deprecated volume types is below:
    - GCEPersistentDisk: https://v1-32.docs.kubernetes.io/docs/concepts/storage/volumes/#gcepersistentdisk
    - AWSElasticBlockStore: https://v1-32.docs.kubernetes.io/docs/concepts/storage/volumes/#awselasticblockstore
    - Glusterfs: https://v1-32.docs.kubernetes.io/docs/concepts/storage/volumes/#glusterfs
  
- `receiver/signalfx`: This receiver is deprecated. Please use the OTLP receiver instead (#46079)
  This component will be removed in a future release.
  
- `receiver/wavefront`: Deprecate the wavefront receiver (#46087)
  There is no replacement for the wavefront receiver.
  Wavefront is EOL: https://support.broadcom.com/web/ecx/support-content-notification/-/external/content/release-announcements/0/25153
  

### 🚀 New components 🚀

- `receiver/vcr`: First PR for new receiver (#42877)

### 💡 Enhancements 💡

- `cmd/golden`: Golden tool now returns all comparison errors with attempt numbers instead of only the last error (#45424)
  When multiple metric payloads fail validation, the golden tool now displays all errors with their attempt numbers,
  making it easier to debug test failures. Previously, only the last error was shown.
  
- `connector/signal_to_metrics`: Add `error_mode` configuration to control OTTL evaluation error handling. Prevents single bad record from failing entire batch. (#38826, #45746)
  Adds three error modes:
  - `propagate` (default): Returns error, entire batch fails (backward compatible)
  - `ignore`: Logs error, skips bad record, continues processing valid records
  - `silent`: Skips bad record silently, continues processing valid records
  
- `exporter/datadog`: Add new alpha feature gate 'exporter.datadogexporter.DisableAllMetricRemapping' to disable all metric remapping in the Datadog exporter. (#45943)
  The feature gate is marked as alpha pending changes in Datadog's backend.
  
- `exporter/elasticsearch`: Update the ECS mode metrics data point hasher to exclude the `elasticsearch.mapping.hints` attribute (#45887)
  Excluding the `elasticsearch.mapping.hints` attribute will allow similar metric data points to be grouped together and indexed to the same document.
- `exporter/elasticsearch`: Add `traces_dynamic_id` config to dynamically set document IDs for traces and span events (#43649)
  Adds `traces_dynamic_id` configuration option to allow setting document IDs based on span and span event attributes using the `elasticsearch.document_id` attribute.
  This prevents duplicate documents from being created when the same span is sent multiple times, similar to the existing `logs_dynamic_id` feature.
  Disabled by default.
  
- `exporter/file`: Add support for rotation when group_by is enabled in file exporter (#43143)
- `exporter/googlecloudstorage`: Add compression support for Google Cloud Storage exporter (#45337)
  The Google Cloud Storage exporter now supports compression of log data before uploading to GCS.
  Supported compression algorithms: `gzip` and `zstd`.
  
- `exporter/kafka`: Add `conn_idle_timeout` configuration option to control when idle connections are not reused and may be closed. (#45321)
  Defaults to 9 minutes.
  
- `extension/awscloudwatchmetricstreams_encoding`: Add support for extracting percentile statistics (p50, p90, p99, etc.) from CloudWatch Metric Streams JSON format (#45855)
  The JSON unmarshaler now extracts percentile fields from CloudWatch Metric Streams data and converts them to OpenTelemetry Summary quantiles.
  This enables feature parity with the embedded cwmetricstream unmarshaler in awsfirehosereceiver.
  
- `extension/awslogs_encoding`: Support CloudWatch Logs extracted fields (`@aws.account`, `@aws.region`) for centralized logging (#45792)
  CloudWatch Logs subscription filter unmarshaler now supports extracted fields (`@aws.account` and `@aws.region`)
  that are automatically added when using CloudWatch Logs centralization and enabling `emitSystemFields` in
  subscription filters. This enhancement enables proper resource attribution in OpenTelemetry when processing
  logs from multiple AWS accounts and regions. Logs with different extracted field values are automatically
  grouped into separate ResourceLogs for proper semantic convention mapping:
  - `@aws.account` maps to `cloud.account.id`
  - `@aws.region` maps to `cloud.region`
  
- `extension/awslogs_encoding`: Handle multiple concatenated JSON objects for AWS CloudWatch Log subscription (#46120)
- `extension/azureauth`: Add and implement new method `Token(context.Context) (*oauth2.Token, error)`. (#45064)
- `extension/encoding`: Introduce streaming support for encoding extensions (#38780)
- `extension/oidc`: Adds `public_keys_file` to the provider config. When set, keys are loaded from a local JWKS file instead of using remote discovery. (#44899)
  The file is watched for changes and keys are automatically reloaded on update. Supported key types are RSA, ECDSA, and ED25519.
  
- `pkg/ottl`: Add `IsInCIDR` function to check if IP belongs to given list of CIDR (#42215)
- `pkg/sampling`: Optimize OTel tracestate parsing by replacing regex validation with hand-written validator (10-21x faster). (#45539)
- `pkg/sampling`: Replace regex-based W3C tracestate validation with hand-written validator for 30-65x performance improvement (#45734)
- `pkg/stanza`: Ensure filter operator does not split batches of entries (#42391)
- `processor/filter`: Introduces inferred context conditions for filtering (#37904)
  Introduces three new top-level config fields [metric_conditions, log_conditions, trace_conditions].
  A user can supply OTTL conditions for each without needing to supply context.
  
- `processor/k8s_attributes`: Added `container.image.tags` resource attribute with feature gate controls according to OpenTelemetry semantic conventions. (#44589)
- `processor/lookup`: Add lookup processor implementation and YAML source (#41816)
  Adds the core lookup processor implementation for enriching telemetry data using external lookups.
  Includes YAML file source for loading lookup tables from local files.
  
- `processor/vultrdetector`: Update semantic conventions to v1.39.0 and add support for cloud.platform in Vultr detector (#45512)
- `receiver/datadog`: Add support for handling the /api/v0.2/stats endpoint to receive and process APM trace stats payloads from the Datadog Agent. (#45778)
  The Datadog Receiver can now process APM trace stats payloads sent by the Datadog Agent via the /api/v0.2/stats endpoint.
  The handler correctly processes gzipped msgpack payloads, decodes them into StatsPayload, translates them into OpenTelemetry-compatible metrics, and forwards them to the configured metrics consumer.
  This enables the complete APM metrics flow: Application → Datadog SDK → Datadog Agent → OpenTelemetry Collector → OTEL Backends.
  
- `receiver/hostmetrics`: Add optional `system.memory.linux.shared` metric (#32712)
  This metric reports shared memory usage, including tmpfs filesystems,
  System V shared memory, and POSIX shared memory. Currently only available
  on Linux systems due to platform-specific data availability.
  This corresponds to the `Shmem` field in `/proc/meminfo`.
  
- `receiver/k8s_cluster`: Add opt-in service metrics derived from k8s Service and EndpointSlice API (#45620)
  New metrics (disabled by default):
  - `k8s.service.endpoint.count`: Number of endpoints by condition (ready, serving, terminating), address type, and zone
  - `k8s.service.load_balancer.ingress.count`: Number of load balancer ingress points assigned to the service
  New resource attributes:
  - `k8s.service.name`: The k8s service name
  - `k8s.service.uid`: The k8s service uid
  - `k8s.service.type`: The k8s service type
  - `k8s.service.traffic_distribution`: The service's traffic routing preference
  - `k8s.service.publish_not_ready_addresses`: Whether the service publishes endpoints before pods are ready
  
- `receiver/kafka`: Add `conn_idle_timeout` configuration option to control when idle connections are not reused and may be closed. (#45321)
  Defaults to 9 minutes.
  
- `receiver/mongodb`: Add support for auth_mechanism, auth_source, and auth_mechanism_properties configuration options (#40686)
  Users can now specify the authentication mechanism (e.g., SCRAM-SHA-256, GSSAPI, MONGODB-AWS), auth source database,
  and auth mechanism properties when connecting to MongoDB instances. This is particularly useful for MongoDB servers
  that require specific authentication mechanisms. For example, GSSAPI (Kerberos) may require SERVICE_NAME, and
  MONGODB-AWS may require AWS_SESSION_TOKEN when using temporary AWS credentials.
  
- `receiver/pprof`: Implement the functionality of transforming pprof to OTel Profiles (#45411)
- `receiver/prometheusremotewrite`: Improved performance when parsing Remote Write v2 requests. (#45623)
- `receiver/prometheusremotewrite`: Add exemplar support to the Prometheus Remote Write receiver (#44983)
- `receiver/redfish`: Change `system.host_name` and `base_url` as resource attributes. (#45470)
- `receiver/sqlquery`: Add support for `initial_delay` in logs collection. (#29671)
  Log collection now applies `initial_delay` (previously ignored). If `initial_delay` is not set, the first log collection now occurs at 1 second, instead of occurring after `collection_interval` time has passed.
  
- `receiver/sqlserver`: Add the `sqlserver.procedure_id` and `sqlserver.procedure_name` attributes to TopQuery and Sample Events (#44656)
  Refined query and reported events to include stored procedure information when applicable. Additionally, the maximum number of active queries reported by default has been increased from 200 to 250 to account for record deaggregation introduced by this change, ensuring the effective limit remains consistent with the previous 200-query baseline.
- `receiver/statsd`: Discard StatsD metrics with NaN or infinite values to prevent invalid data from entering the metric pipeline (#44288)
- `receiver/syslog`: Add facility_text attribute to syslog parser output (#45641)
  The syslog parser now outputs a facility_text attribute containing
  the human-readable facility name (e.g., "auth", "kern", "local0")
  in addition to the existing numeric facility attribute.
  

### 🧰 Bug fixes 🧰

- `exporter/datadog`: OTLP logs now support array type attributes. Arrays containing primitive values or nested maps are now correctly preserved in the log output. (#45708)
- `exporter/datadog`: Fix data race in the Datadog exporter which could cause a crash with error message "concurrent map iteration and map write". (#46051)
  Specifically, when processing spans with the `datadog.host.use_as_metadata` attribute.
  
- `exporter/elasticsearch`: Fix ECS mode to properly protect known schema fields from getting `.value` suffix when conflicting with nested attributes (#37211)
  Previously, when ECS mode was enabled and attributes like `process.executable.name` conflicted with the known ECS field `process.executable`, the deduplication logic would incorrectly add a `.value` suffix to the known field, resulting in `process.executable.value`. This fix ensures protected ECS fields remain unchanged and conflicting nested attributes are properly ignored.
  
- `exporter/opensearch`: Fix `sending_queue` not using default values for `num_consumers` and `queue_size` when only `batch` is configured (#45016)
- `exporter/syslog`: Update the timestamp when using the RFC 3164 formatter to space-pad the day of month for single digit days (#46115)
- `extension/awslogs_encoding`: Fix duplicate resource attributes in subscription filter unmarshaler (#45792)
  The `aws.log.group.names` and `aws.log.stream.names` resource attributes were incorrectly
  being set twice: first as array values, then immediately overwritten as string values.
  This fix removes the duplicate string assignments, ensuring the attributes are correctly
  set only as arrays per OpenTelemetry semantic conventions.
  
- `extension/oauth2client`: Fix oauth2clientauth client-credentials grant type (#45786)
- `extension/text_encoding`: Fix text encoding extension to not split large messages when no separator is configured. (#45845)
- `pkg/stanza`: Fix recombine operator logging errors at ERROR level when `on_error` is set to quiet mode (#42646)
- `pkg/translator/prometheusremotewrite`: Fix export of Instrumentation Scope attributes as Prometheus labels. (#45912)
  Instrumentation Scope attributes (name, version, and other attributes) are now correctly translated to Prometheus labels with the `otel_scope_` prefix.
- `pkg/xk8stest`: Fix IPv6 gateway handling in HostEndpoint to avoid invalid address formatting in e2e tests (#46082)
  Prefer IPv4 gateways when resolving the Docker kind network gateway.
  Fall back to bracketed IPv6 if no IPv4 gateway is found, so that
  appending :port produces a valid address (e.g. [::1]:4317).
  
- `processor/k8s_attributes`: Fix concurrent map access panic by cloning pod labels and annotations before extraction. (#46112)
- `processor/k8s_attributes`: Allow key_regex to work without tag_name by using the default tag name format (#45719)
  When using key_regex with capturing groups but without specifying tag_name, the processor now
  correctly uses the default tag name format (e.g., k8s.pod.labels.<label_key>) instead of
  producing empty tag names.
  
- `processor/redaction`: Improve database sanitization with system-aware obfuscation, span name sanitization, and URL path parameter redaction. (#44229)
  - Database sanitization now validates span kind (CLIENT/SERVER/INTERNAL ) and requires db.system.name/db.system attribute for traces/metrics
  - Implemented span name obfuscation for database operations based on db.system
  - Added URL path parameter sanitization for span names with configurable pattern matching
  - Improved query validation database sanitizers
  - Fix issue ensuring no spans with `...` name can be generated due to enabling multiple sanitizers
  - If something went wrong during span name sanitization, original span name is used
  
- `receiver/azure_event_hub`: Fixes a bug where the receiver would stop receiving messages after a parsing error. (#45898)
- `receiver/faro`: Updates Faroreceiver to return HTTP 202 Accepted status code upon successful data ingestion to comply with the OpenAPI specification. (#45648)
- `receiver/fluentforward`: handle uint64 to int64 overflow by changing to string (#45252)
  FluentD supports record entries with uint64 types. OpenTelemetry log attributes only support int64 and no uint64.
  The old solution would overflow with uint64 values greater than `math.MaxInt64` and result in negative attribute values.
  This fix changes that behaviour by storing only those large values as string attributes instead.
  
- `receiver/googlecloudpubsub`: Fix compression detection when both encoding and compression are set in the config (#45810)
- `receiver/mongodb`: Check if metrics are enabled before collecting them to prevent errors when metrics are disabled. (#41465)
- `receiver/postgresql`: Updated the default value for top_n_query (200) to match with other db receivers (#45612)

<!-- previous-version -->


## v0.146.0

### 🛑 Breaking changes 🛑

- `all`: Increase minimum Go version to 1.25 (#14567)

### 🚩 Deprecations 🚩

- `pdata/pprofile`: Declare removed aggregation elements as deprecated. (#14528)

### 💡 Enhancements 💡

- `all`: Add detailed failure attributes to exporter send_failed metrics at detailed telemetry level. (#13956)
  The `otelcol_exporter_send_failed_{spans,metric_points,log_records}` metrics now include
  failure attributes when telemetry level is Detailed: `error.type` (OpenTelemetry semantic convention
  describing the error class) and `error.permanent` (indicates if error is permanent/non-retryable).
  The `error.type` attribute captures gRPC status codes (e.g., "Unavailable", "ResourceExhausted"),
  standard Go context errors (e.g., "canceled", "deadline_exceeded"),
  and collector-specific errors (e.g., "shutdown").
  This enables better alerting and debugging by providing standardized error classification.
- `cmd/builder`: Introduce new experimental `init` subcommand (#14530)
  The new `init` subcommand initializes a new custom collector
- `cmd/builder`: Add "telemetry" field to allow configuring telemetry providers (#14575)
  Most users should not need to use this, this field should only be set if you
  intend to provide your own OpenTelemetry SDK.
  
- `cmd/mdatagen`: Introduce additional metadata (the version since the deprecation started, and the deprecation reason) for deprecated metrics. (#14113)
- `cmd/mdatagen`: Add optional `relationships` field to entity schema in metadata.yaml (#14284)
- `exporter/debug`: Add `output_paths` configuration option to control output destination when `use_internal_logger` is false. (#10472)
  When `use_internal_logger` is set to `false`, the debug exporter now supports configuring the output destination via the `output_paths` option.
  This allows users to send debug exporter output to `stdout`, `stderr`, or a file path.
  The default value is `["stdout"]` to maintain backward compatibility.
  
- `pkg/confmap`: Add experimental `ToStringMapRaw` function to decode `confmap.Conf` into a string map without losing internal types (#14480)
  This method exposes the internal structure of a `confmap.Conf` which may change at any time without prior notice
  

### 🧰 Bug fixes 🧰

- `cmd/mdatagen`: Reset aggDataPoints during metric init to avoid index out of range panic across emit cycles when reaggregation is enabled. (#14569)
- `cmd/mdatagen`: Fix panic when mdatagen is run without arguments. (#14506)
- `pdata/pprofile`: Fix off-by-one issue in dictionary lookups. (#14534)
- `pkg/config/confighttp`: Fix high cardinality span name from request method from confighttp server internal telemetry (#14516)
  Follow spec to bound request method cardinality.
- `pkg/otelcol`: Ignore component aliases in the `otelcol components` command (#14492)
- `pkg/otelcol`: Order providers and converters in alphabetical order in the `components` subcommand. (#14476)

<!-- previous-version -->
