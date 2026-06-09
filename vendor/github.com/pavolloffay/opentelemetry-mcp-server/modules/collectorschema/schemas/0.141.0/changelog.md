## v0.141.0

### 🛑 Breaking changes 🛑

- `all`: fix pprofile DurationNano to be a TypeUint64 (#44397)
- `cmd/otelcontribcol`: Removing unmaintained component `exporter/carbon` (#38913)
- `connector/spanmetrics`: Add a feature gate to use the latest semantic conventions for the status code attribute on generated metrics. | This feature gate will replace the `status.code` attribute on the generated RED metrics with `otel.status_code`. | It will also replace the values `STATUS_CODE_ERROR` and `STATUS_CODE_OK` with `ERROR` and `OK` to align with the latest conventions. (#42103)
  This change is made to align with [the latest semantic conventions](https://opentelemetry.io/docs/specs/semconv/registry/attributes/otel/#otel-status-code). | The feature gate is disabled by default, but can be enabled with `--feature-gates spanmetrics.statusCodeConvention.useOtelPrefix` | or explicitly disabled with `--feature-gates -spanmetrics.statusCodeConvention.useOtelPrefix`.
- `exporter/clickhouse`: Add EventName column to logs table (#42584)
  This column is optional for existing deployments. See project README for notes on how to upgrade your logs table.
- `exporter/clickhouse`: Add columns for tracking JSON paths in logs + traces (#43109)
  The JSON columns now include a helper column for keeping track of what keys are inside of the JSON object.
  This change also introduces schema detection logic to reduce breaking changes whenever a column is added.
  Existing users can enable these features by manually adding all the new columns to their table.
  
- `exporter/kafka`: `exporter.kafkaexporter.UseFranzGo` feature gate moved to Stable and is now always enabled (#44565)
  The franz-go client is now the default and only Kafka client library for the Kafka exporter.
  The feature gate `exporter.kafkaexporter.UseFranzGo` has been promoted to Stable status and cannot be disabled.
  Users can no longer opt out of using the franz-go client in favor of the legacy Sarama client.
  The Sarama client and the feature gate will be removed completely after v0.143.0.
  
- `extension/docker_observer`: Upgrading Docker API version default from 1.24 to 1.44 (#44279)
- `pkg/ottl`: Type of field profile.duration changes from time.Time to int64. (#44397)
- `receiver/azureeventhub`: Promote Feature Gate `receiver.azureeventhubreceiver.UseAzeventhubs` to Beta (#44335)
- `receiver/k8slog`: Update k8slogreceiver code-owners status and mark as unmaintained (#44078)
- `receiver/kafka`: Remove deprecated topic and encoding (#44568)
- `receiver/kafka`: `receiver.kafkareceiver.UseFranzGo` feature gate moved to Stable and is now always enabled (#44564)
  The franz-go client is now the default and only Kafka client library for the Kafka receiver.
  The feature gate `receiver.kafkareceiver.UseFranzGo` has been promoted to Stable status and cannot be disabled.
  Users can no longer opt out of using the franz-go client in favor of the legacy Sarama client.
  The Sarama code and the feature gate will be removed completely after v0.143.0.
  

### 🚩 Deprecations 🚩

- `receiver/prometheus`: Add feature gate for extra scrape metrics in Prometheus receiver (#44181)
  deprecation of extra scrape metrics in Prometheus receiver will be removed eventually.

### 🚀 New components 🚀

- `connector/metricsaslogs`: Add connector to convert metrics to logs (#40938)
- `extension/azure_encoding`: [extension/azure_encoding] Introduce new component (#41725)
  This change includes only overall structure, readme and configuration for a new component
- `receiver/awslambda`: Implementation of the AWS Lambda Receiver. (#43504)
- `receiver/macosunifiedlogging`: Add a new receiver for macOS Unified Logging. (#44089)
- `receiver/macosunifiedlogging`: Implementation of the macOS Unified Logging Receiver. (#44089)

### 💡 Enhancements 💡

- `connector/count`: Support all attribute types in the count connector (#43768)
- `connector/routing`: Avoid extra copy of all data during routing (#44387)
- `exporter/awss3`: Support compression with ZSTD (#44542)
- `exporter/azuremonitor`: Add additional mapping of standard OTel properties to builtin Application Insights properties (#40598)
- `exporter/cassandra`: `cassandraexporter`: Upgrade cassandra library version (#43691)
  Upgrade cassandra library version.
  
- `exporter/elasticsearch`: Updates the ecs mode span encode to include the `span.kind` attribute (#44139)
- `exporter/elasticsearch`: add missing fields to struct so that they are populated in the respective elasticsearch index (#44234)
- `exporter/file`: Add create_directory and directory_permissions options; exporter can automatically create parent directories (also honored by group_by) with configurable permissions. (#44280)
  - New config: `create_directory` (bool) and `directory_permissions` (octal string, e.g. \"0755\").
  - When enabled, the exporter creates the parent directory of `path` on start.
  - `group_by` uses the configured permissions when creating per-attribute directories.
  
- `exporter/googlecloudpubsub`: Update to cloud.google.com/go/pubsub/v2. (#44465)
- `exporter/googlecloudpubsub`: Add encoding extension support (#42270, #41834)
  Add encoding extension support for the payload on Pub/Sub. As having custom extensions means the Pub/Sub attributes
  cannot be auto discovered additional functionality has been added to set the message attributes.
  
- `exporter/prometheus`: Add without_scope_info to omit otel scope info from prometheus exporter metrics (#43613)
- `exporter/prometheus`: Add support to exponential histograms (#33703)
- `exporter/signalfx`: Makes sending tags from SignalFx Exporter configurable (#43799)
  New optional configuration flag `drop_tags` has been added to SignalFx Exporter to allow users to disable tag metadata sending.
  This feature has been introduced due to a common issue among Splunk Observability customers when they're receiving more tags 
  than allowed limit. 
  
- `extension/awslogs_encoding`: Add more fields to AWS NLB logs at ELB extension (#43757)
- `extension/googlecloudlogentry_encoding`: Add support for Proxy Network Load Balancer logs (#44099)
  Add support for Proxy Network Load Balancer logs to the Google Cloud log entry encoding extension.
  This includes adding support for the new `gcp.load_balancing.proxy_nlb` attributes.
  See the [README](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/extension/encoding/googlecloudlogentryencodingextension/README.md#proxy-network-load-balancer-logs) for more details.
  
- `extension/headers_setter`: Add support for chaining with other auth extensions via `additional_auth` configuration parameter. This allows combining multiple authentication methods, such as OAuth2 for bearer token authentication and custom headers for additional metadata. (#43797)
  The `additional_auth` parameter enables the `headers_setter` extension to work in conjunction
  with other authentication extensions like `oauth2client`. The additional auth extension is called
  first to apply its authentication, then headers_setter adds its configured headers on top.
  
- `extension/observer`: Add container name, container ID, and container image to port endpoint (#41309)
- `pkg/ottl`: Add `ParseSeverity` function to define mappings for log severity levels. (#35778)
- `pkg/ottl`: Introduce `CommunityID` function to generate network hash (20-byte SHA1 digest) flow from the given source and destination {IP, port}, optionally protocol and seed values. (#34062)
- `pkg/ottl`: Expand usage of literal into typed getters and avoid unnecessary work (#44201)
- `pkg/ottl`: SliceToMap: add support to convert slices with non-map elements to maps (#43099)
- `processor/cumulativetodelta`: Add support for exponential histograms (#44106)
- `processor/resourcedetection`: Use `osProfile.computerName` for setting `host.name` in Azure resource detection processor (#43959)
- `processor/resourcedetectionprocessor/oraclecloud`: Oracle Cloud (OCI) resource detection processor now differentiates between running off-platform (expected not to work), and encountering an error when running on-platform (expected to work) (#42794)
  - Collectors not running on Oracle Cloud return an empty resource and no error, since we don't expect this case to work.
  - If the Oracle Cloud platform is detected but metadata fetch fails, the error is now logged and signaled to the processor, since we do expect this case to work.
  
- `processor/tail_sampling`: Add bytes_limiting policy type, sample based on the rate of bytes per second using a token bucket algorithm. (#42509)
- `processor/tail_sampling`: Adaptive expected_new_traces_per_sec to improve performance lower bound (#43561)
- `receiver/googlecloudpubsub`: Update to cloud.google.com/go/pubsub/v2. (#44466)
- `receiver/googlecloudpubsub`: Adjusts the subscription regex to accommodate new project naming used for Google Trusted Partner Clouds. (#43988)
- `receiver/googlecloudpubsubpush`: Add telemetry metrics to the component. (#44422)
- `receiver/googlecloudpubsubpush`: Add implementation to googlecloudpubsubpush receiver. (#44101)
- `receiver/k8s_events`: Allow more event types like Error and Critical which are typically used by applications when creating events. (#43401)
  k8seventsreceiver allows event types Error and Critical in addition to the current Normal and Warning event types.
- `receiver/kafka`: Add support for exclude topics when consuming topics with a regex pattern (#43782)
- `receiver/prometheus`: Support JWT Profile for Authorization Grant (RFC 7523 3.1) (#44381)
- `receiver/redis`: Add support for redis.mode and redis.sentinel.* metrics (#42365)
- `receiver/systemd`: Promote systemd receiver to alpha (#33532)
- `receiver/systemd`: Scrape unit CPU time (#44646)

### 🧰 Bug fixes 🧰

- `cmd/opampsupervisor`: Fix supervisor passthrough logs overflow by using bufio.Reader instead of bufio.Scanner (#44127)
- `cmd/opampsupervisor`: Fix data race in `remoteConfig` field by using atomic pointer for thread-safe concurrent access (#44173)
- `connector/routing`: Fix routing to default route when error occurs (#44386)
  Before we used to send everything (even records match without error) to the default pipeline, |
  after this change only entries that return error will be "ignored" and if no other rule in the |
  table picks them will be sent to the default rule.
  
- `exporter/clickhouse`: Fix TLS configuration being ignored when only ca_file is provided and no cert/key files are set. (#43911)
  This change ensures server-side TLS validation works correctly even without client certificates.
- `exporter/elasticsearch`: Fix CloudID parsing to correctly handle Elastic Cloud IDs when sent with multiple dollar sign separators (#44306)
  The CloudID decoder was incorrectly using `strings.Cut()` which only splits on the first delimiter,
  causing malformed URLs when the decoded CloudID contained multiple `$` separators. Changed to use
  `strings.Split()` to match the reference implementation from go-elasticsearch library.
  
- `extension/awslogs_encoding`: address the SIGSEGV occurring when processing control_message messages. (#44231)
- `extension/awslogs_encoding`: Fix ALB log `request_line` parsing for valid formats and avoid errors (#44233)
- `pkg/ottl`: Fixed OTTL grammar to treat the string literal "nil" as ordinary text instead of a nil value. (#44374)
- `pkg/ottl`: Return errors when OTTL context setters receive values of the wrong type (#40198)
  Introduces `ctxutil.ExpectType` and updates log, metric, and scope setters to surface type assertion failures.
  
- `pkg/ottl`: Fix TrimPrefix/TrimSuffix function name. (#44630)
  This change also adds a featuregate "ottl.PanicDuplicateName" to control the behavior of panicing when duplicate
  names are registered for the same function.
  
- `processor/k8sattributes`: `k8sattributesprocessor` now respects semantic convention resolution order for `service.namespace` (#43919)
  Previously, when `service.namespace` was included in the extract metadata configuration, the processor
  would incorrectly allow `k8s.namespace.name` to override explicitly configured service namespace values
  from OpenTelemetry annotations (e.g., `resource.opentelemetry.io/service.namespace`). Now the processor
  correctly follows the semantic convention resolution order, where annotation values take precedence over
  inferred Kubernetes namespace names.
  
- `processor/k8sattributes`: Fix incorrect pod metadata assignment when `host.name` contains a non-IP hostname (#43938)
  The processor now correctly validates that `host.name` contains an IP address before using it for pod association.
  Previously, textual hostnames were incorrectly used for pod lookups, causing spans and metrics from one workload
  to receive metadata from unrelated pods that shared the same hostname.
  
- `receiver/awsxray`: Fix incorrect span kind when translating X-Ray segment to trace span with parent ID (#44404)
- `receiver/azuremonitor`: Collect only supported aggregations for each metric (501 not implemented issue) (#43648)
  Some metrics were not collected because we requested all available aggregation types. This led to 501 errors, as the Azure API returned responses indicating that certain aggregations were not implemented.
  We now use the supported aggregations field from each metric definition to filter and request only the aggregations that are actually supported.
  The user can expect less 501 errors in the logs and more metrics in the results.
  
- `receiver/datadog`: Utilizes thread safe LRU packages (#42644)
- `receiver/github`: Adds corrections to span times when GitHub sends incorrect start and end times. (#43180)
- `receiver/libhoney`: Allow single events and uncompressed requests (#44026, #44010)
  The receiver required events to be wrapped in an array before. The single-event format
  was allowed by Honeycomb's API so we have added it here.
  This fix also allows uncompressed requests again
  
- `receiver/sqlquery`: Fix a bug in the sqlqueryreceiver where an error is returned if the query returned a null value. This is now logged as a warning and logs with null values are ignored. (#43984)
- `receiver/systemd`: This allows systemd receiver to be used in collector config (#44420)

<!-- previous-version -->


## v1.47.0/v0.141.0

### 🛑 Breaking changes 🛑

- `pkg/config/confighttp`: Use configoptional.Optional for confighttp.ClientConfig.Cookies field (#14021)

### 💡 Enhancements 💡

- `pkg/config/confighttp`: Setting `compression_algorithms` to an empty list now disables automatic decompression, ignoring Content-Encoding (#14131)
- `pkg/service`: Update semantic conventions from internal telemetry to v1.37.0 (#14232)
- `pkg/xscraper`: Implement xscraper for Profiles. (#13915)

### 🧰 Bug fixes 🧰

- `pkg/config/configoptional`: Ensure that configoptional.None values resulting from unmarshaling are equivalent to configoptional.Optional zero value. (#14218)

<!-- previous-version -->
