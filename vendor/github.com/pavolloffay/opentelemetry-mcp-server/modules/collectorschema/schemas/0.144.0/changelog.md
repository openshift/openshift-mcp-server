## v0.144.0

### 🛑 Breaking changes 🛑

- `exporter/elasticsearch`: Remove ecs mode span enrichment for `span.action`, `span.message.queue.name`, and `transaction.message.queue.name` (#45014)
  The removed span enrichments have been moved to the `github.com/elastic/opentelemetry-collector-components/processor/elasticapmprocessor`. It is recommended to use the `github.com/elastic/opentelemetry-collector-components/processor/elasticapmprocessor` when using mapping mode `ecs` to ensure index documents contain all required Elastic fields to power the Kibana UI.
- `exporter/kafka`: Remove Sarama producer implementation (#44565)
  The Sarama-based Kafka producer has been removed from kafkaexporter.
  Feature gate `exporter.kafkaexporter.UseFranzGo` has also been removed since Franz-go is now the only supported Kafka client.
  
- `processor/tail_sampling`: The deprecated invert decisions are disabled by default. (#44132)
  Drop policies should be used instead of invert decisions for explicitly not sampling a trace.
  If the deprecated behavior is required while migrating to drop policies, disable the `processor.tailsamplingprocessor.disableinvertdecisions` feature gate.
  
- `receiver/kafka`: Remove Sarama consumer implementation and `default_fetch_size` configuration option (#44564)
  The Sarama-based Kafka consumer has been removed from kafkareceiver.
  The `default_fetch_size` configuration option has also been removed as it was only used by the Sarama consumer.
  Feature gate `receiver.kafkareceiver.UseFranzGo` has also been removed since Franz-go is now the only supported Kafka client.
  

### 🚩 Deprecations 🚩

- `exporter/elasticsearch`: Deprecate `mapping::mode` config option (#45246)
  The `mapping::mode` config option is now deprecated and will soon be ignored. Instead, use
  the `X-Elastic-Mapping-Mode` client metadata key (via headers_setter extension) or the
  `elastic.mapping.mode` scope attribute to control the mapping mode per-request.
  See the README for migration instructions.
  

### 🚀 New components 🚀

- `processor/lookup`: Add skeleton for external lookup enrichment processor (#41816)
  Adds the initial skeleton for a lookup processor that performs external lookups to enrich telemetry signals.
  Also includes source abstraction with factory pattern, noop source for testing, and cache wrapper utility.
  

### 💡 Enhancements 💡

- `cmd/schemagen`: Extend schemagen script with ability to handle external refs. (#42214)
  The `schemagen` tool has been enhanced to support external references when generating
  configuration schemas for OpenTelemetry Collector components. This improvement allows
  the tool to accurately reference and include schema definitions from external packages,
  facilitating better modularity and reuse of configuration schemas across different components.
  
- `cmd/schemagen`: Fixes for schemagen to handle common issues with receiver components schemas. (#42214)
  Fix common issues discovered while using schemagen with receiver components:
  
  - Missing config.go file (e.g. namedpipereceiver)
  - Parsing obsolete types (e.g. nsxtreceiver)
  - Unable to embed fields with `squash` tag and not exported internal type (e.g. huaweicloudcesreceiver)
  
- `cmd/telemetrygen`: Add batching capability to metrics and traces (#42322)
  - Changed traces batching to have configurable batch size and match batch flag.
  - Added batching to metrics.
  - Added batching to logs. 
  
- `exporter/azureblob`: Add timezone option for formatting blob names in azureblob exporter. (#43752)
- `exporter/elasticsearch`: Remove go-elasticsearch dependency to reduce binary size (#45104)
  This leads to a 19MB size reduction in contrib distribution
- `exporter/googlecloudstorage`: Add support for time partitioning (#44889)
- `exporter/opensearch`: Add support for multiple variables to build index names (#42585)
- `exporter/sumologic`: Add `decompose_otlp_summaries` configuration option to decompose OTLP Summary metrics into individual gauges and counters (#44737)
- `extension/awslogs_encoding`: Optimize CloudTrail logs unmarshaling for memory usage (#45180)
- `processor/k8sattributes`: Bumnp version of semconv to v1.39.0 (#45447)
- `processor/redaction`: Add `sanitize_span_name` option to URL and DB sanitization configs. (#44228)
- `processor/redaction`: Add `ignored_key_patterns` configuration option to allow ignoring keys by regex pattern (#44657)
- `processor/resourcedetection`: Add optional docker attributes (#44898). **Note**: Because of [opentelemetry-collector-releases#1350](https://github.com/open-telemetry/opentelemetry-collector-releases/issues/1350) this change is not available on the v0.144.0 binary releases. v0.145.0 will include this change.
  Add `container.image.name` and `container.name` optional resource attributes with the docker detector.
  
- `processor/tail_sampling`: Provide an option, `decision_wait_after_root_received`, to make quicker decisions after a root span is received. (#43876)
- `receiver/azureeventhub`: Add support for azure auth when feature gate `receiver.azureeventhubreceiver.UseAzeventhubs` is enabled. (#40711)
- `receiver/prometheus`: receiver/prometheus now associates scraped _created text lines as the created timestamp of its metric family rather than its own metric series, as defined by the OpenMetricsText spec (#45291)
- `receiver/prometheus`: Add comprehensive troubleshooting and best practices guide to Prometheus receiver README (#44925)
  The guide includes common issues and solutions, performance optimization strategies,
  production deployment best practices, monitoring recommendations, and debugging tips.
  
- `receiver/prometheusremotewrite`: Replace labels.Map() iteration with direct label traversal to eliminate intermediate map allocations. (#45166)

### 🧰 Bug fixes 🧰

- `exporter/kafka`: franz-go: Exclude non-produce metrics from kafka_exporter_write_latency and kafka_exporter_latency (#45258)
- `exporter/opensearch`: Fix dynamic log index feature putting logs in wrong index (#43183)
- `exporter/prometheusremotewrite`: Prevent duplicate samples by allowing the WAL to be empty (#41785)
  Since the WAL is being truncated after every send it's likely the reader and writer are in sync. Since WAL was not
  allowed to be empty, the reader would always re-read previously delivered samples causing duplicate data to be sent
  continuously.
  
- `extension/datadog`: Datadog extension no longer throws an error for missing extensions when getting a list of active components, and now populates active components even when missing go mod/version info. (#45358, #45460)
- `extension/file_storage`: Handle filename too long error in file storage extension by using the sha256 of the attempted filename instead. (#44039)
- `extension/text_encoding`: Avoid spurious marshalling separators at end of lines (#42797)
  Previously, text_encoding would append the marshalling separator to the end of
  each log record, potentially resulting in double-newlines between blocks of
  records.
  
- `extension/text_encoding`: Fix an issue where marshalling/unmarshalling separators were ignored (#42797)
- `pkg/kafka/configkafka`: Fix consumer group rebalance strategy validation (#45268)
- `pkg/ottl`: Fix numeric parsing to correctly handle signed numbers in math expressions. (#45222)
  The OTTL math expression parser did not correctly handle unary signs for plus
  and minus. Expressions like `3-5` would not parse correctly without inserting
  spaces to make it `3 - 5`. This change moves the sign handling out of the
  lexer and into the parser.
  
- `pkg/ottl`: Handle floating constants with decimal point but no fraction. (#45222)
  Floating point constants that had a decimal point but no fractional digits
  (e.g., "3.") were not handled properly and could crash the parser. These are
  now parsed as valid floating point numbers.
  
- `pkg/stanza`: Fix Windows UNC network path handling in filelog receiver (#44401)
  The filelog receiver now correctly handles Windows UNC network paths (e.g., \\server\share\logs\*.log).
  Previously, the receiver could list files from network shares but failed to open them due to path corruption
  during normalization. This fix converts UNC paths to Windows extended-length format (\\?\UNC\server\share\path)
  which is more reliable and not affected by filepath.Clean() issues.
  
- `pkg/stanza`: Ensure `container` parser respects the `if` condition and `on_error` settings when format detection fails (#41508)
- `processor/resourcedetection`: Prevent the resource detection processor from panicking when detectors return a zero-valued pdata resource. (#41934) **Note**: Because of [opentelemetry-collector-releases#1350](https://github.com/open-telemetry/opentelemetry-collector-releases/issues/1350) this change is not available on the v0.144.0 binary releases. v0.145.0 will include this change.
- `processor/resourcedetection`: Fix nil pointer panic when HTTP client creation fails in Start method (#45220) **Note**: Because of [opentelemetry-collector-releases#1350](https://github.com/open-telemetry/opentelemetry-collector-releases/issues/1350) this change is not available on the v0.144.0 binary releases. v0.145.0 will include this change.
- `receiver/awslambda`: Fix S3 key usage in AWS Lambda Receiver (#45364)
- `receiver/datadog`: Fix service check endpoint to handle both array and single object payloads (#44986)
  The `/api/v1/check_run` endpoint now uses defensive parsing to handle both array `[{...}]` and single object `{...}` payloads.
  This fixes intermittent unmarshal errors when Datadog agent sends connectivity health checks.
  
- `receiver/jmx`: Enable initial_delay and collection_interval settings via scraper helper (#44492)
- `receiver/libhoney`: Improve msgpack decoding to handle ints or uints (#45273)
- `receiver/postgresql`: Fix query plan EXPLAIN to use raw query with $N placeholders instead of obfuscated query with ? placeholders (#45190)
  Previously, the EXPLAIN query was using obfuscated queries with ? placeholders, which PostgreSQL does not recognize.
  Now uses the raw query with $1, $2 placeholders that PostgreSQL expects.
  
- `receiver/prometheusremotewrite`: Fix silent data loss when consumer fails by returning appropriate HTTP error codes instead of 204 No Content. (#45151)
  The receiver was sending HTTP 204 No Content before calling ConsumeMetrics(),
  causing clients to believe data was successfully delivered even when the consumer failed.
  Now returns 400 Bad Request for permanent errors and 500 Internal Server Error for retryable errors,
  as per the Prometheus Remote Write 2.0 specification.
  
- `receiver/sqlserver`: Accuracy improvements for top-query metrics (#45228)
  SQLServer metrics reporting is improved by reducing the warm-up delay and providing accurate insights sooner.
  

<!-- previous-version -->


## v1.50.0/v0.144.0

### 🛑 Breaking changes 🛑

- `pkg/exporterhelper`: Change verbosity level for otelcol_exporter_queue_batch_send_size metric to detailed. (#14278)
- `pkg/service`: Remove deprecated `telemetry.disableHighCardinalityMetrics` feature gate. (#14373)
- `pkg/service`: Remove deprecated `service.noopTracerProvider` feature gate. (#14374)

### 🚩 Deprecations 🚩

- `exporter/otlp_grpc`: Rename `otlp` exporter to `otlp_grpc` exporter and add deprecated alias `otlp`. (#14403)
- `exporter/otlp_http`: Rename `otlphttp` exporter to `otlp_http` exporter and add deprecated alias `otlphttp`. (#14396)

### 💡 Enhancements 💡

- `cmd/builder`: Avoid duplicate CLI error logging in generated collector binaries by relying on cobra's error handling. (#14317)
- `cmd/mdatagen`: Add the ability to disable attributes at the metric level and re-aggregate data points based off of these new dimensions (#10726)
- `cmd/mdatagen`: Add optional `display_name` and `description` fields to metadata.yaml for human-readable component names (#14114)
  The `display_name` field allows components to specify a human-readable name in metadata.yaml.
  When provided, this name is used as the title in generated README files.
  The `description` field allows components to include a brief description in generated README files.
  
- `cmd/mdatagen`: Validate stability level for entities (#14425)
- `pkg/xexporterhelper`: Reenable batching for profiles (#14313)
- `receiver/nop`: add profiles signal support (#14253)

### 🧰 Bug fixes 🧰

- `pkg/exporterhelper`: Fix reference count bug in partition batcher (#14444)

<!-- previous-version -->
