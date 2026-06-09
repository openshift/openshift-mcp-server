## v0.137.0

### 🛑 Breaking changes 🛑

- `spanmetricsconnector`: Exclude all resource attributes in spanmetrics (#42103)
  This change aligns with the ServiceGraph implementation and may introduce a breaking change:
  
  - Users utilizing Prometheus remote write will not experience a breaking change.
  - Users using OTLP/HTTP may encounter a breaking change.
  
  The change is currently guarded by the feature gate `connector.spanmetrics.excludeResourceMetrics` and is disabled by default. 
  It will be enabled by default in the next release.
  
- `spanmetricsconnector`: Change default duration metrics unit from `ms` to `s` (#42462)
  This change introduces a breaking change, which is now guarded by the feature gate `connector.spanmetrics.useSecondAsDefaultMetricsUnit`.
  Currently, the feature gate is disabled by default, so the unit will remain `ms`. After one release cycle, the unit will switch to `s` and the feature gate will also be enabled by default.
  
- `splunkhecexporter`: Removed deprecated `HecToOtelAttrs` configuration from splunkhecexporter (#43005)
- `signalfxreceiver`: Removed deprecated configuration option: access_token_passthrough (#43048)
  As access token passthrough option is no longer supported, to achieve similar behavior configure your collector
  to use the `headers_setter` extension to pass the access token.
  
- `stefexporter, stefreceiver`: Update to STEF 0.0.8. This is a breaking change in protocol format. (#42949)
  Make sure to update both ends (receiver and exporter) to the same STEF version.

### 🚩 Deprecations 🚩

- `awslogsencodingextension`: Rename awslogsencodingextension format values to shorter, more concise identifiers. Old format values are still supported until v0.138.0. (#42901)
- `pkg/datadog, exporter/datadog, extension/datadog`: Deprecates StaticAPIKeyCheck, stops doing validation for API key characters in Datadog exporter and extension. (#42677)
  This was causing issues to users since validation of secrets is challenging
  

### 🚀 New components 🚀

- `googlecloudstorageexporter`: Add skeleton of googlecloudstorage exporter (#42137)
- `receiver/ciscoosreceiver`: Add initial skeleton of Cisco OS receiver (README, config, factory, metadata) with In Development stability. (#42647)
  This PR adds structure only (no scraping implementation yet).
  Scrapers and SSH-based collection logic (BGP, Environment, Facts, Interfaces, Optics) will be added in follow-up PRs.
  
- `unrollprocessor`: Adds a processor that generically takes a log body of slices and creates new entries from that slice. (#42491)
- `resourcedetectionprocessor`: Added Oracle Cloud resource detection support to resourcedetectionprocessor, enabling automatic population of Oracle Cloud-specific resource attributes. (#35091)
  This update allows the OpenTelemetry Collector to detect and annotate telemetry with Oracle Cloud resource metadata when running in Oracle Cloud environments.
  Includes new unit tests and documentation.
  

### 💡 Enhancements 💡

- `redactionprocessor`: Add support for URL sanitization in the redaction processor. (#41535)
- `unrollprocessor`: Bump the stability to Alpha, and include it in otelcontribcol (#42917)
- `awscloudwatchlogsexporter`: Adding yaten2302 as code owner for awscloudwatchlogsexporter, move it from unmaintained to alpha (#43039)
- `coralogixexporter`: Add Automatic AWS PrivateLink set up via new `private_link` configuration option (#43075)
  When enabled, the exporter will automatically use the AWS PrivateLink endpoint for the configured domain.
  If the domain is already set to a PrivateLink one, no further change to the endpoint will be made.
  
- `receiver/kafkametricsreceiver`: Add support for using franz-go client under a feature gate (#41480)
- `receiver/k8seventsreceiver`: Added support for Leader Election into `k8seventsreceiver` using `k8sleaderelector` extension. (#42266)
- `receiver/k8sobjectsreceiver`: Switch to standby mode when leader lease is lost instead of shutdown (#42706)
- `kafkareceiver`: Add `max_partition_fetch_size` configuration option to kafkareceiver (#43097)
- `processor/resourcedetection`: Add support for DigitalOcean in resourcedetectionprocessor (#42803)
- `processor/resourcedetection`: Add support for upcloud in resourcedetectionprocessor (#42801)
- `receiver/kafka`: Add support for disabling KIP-320 (truncation detection via leader epoch) for Franz-Go (#42226)
- `haproxyreceiver`: Add support for act, weight, ctime, qtime, rtime, bck and slim metrics from HAProxy (#42829)
- `hostmetricsreceiver`: Add useMemAvailable feature gate to use the MemAvailable kernel's statistic to compute the "used" memory usage (#42221)
- `otlpencodingextension`: Promote the otlpencodingextension extension to beta. (#41596)
- `receiver/kafkareceiver`: Use franz-go client for Kafka receiver as default, promoting the receiver.kafkareceiver.UseFranzGo feature gate to Beta. (#42155)
- `oracledbreceiver`: Add `service.instance.id` resource attribute (#42402)
  The `service.instance.id` resource attribute is added in the format `<host>:<port>/<service>` to uniquely identify 
  Oracle DB hosts. This resource attribute is enabled by default for metrics and logs.
  
- `extension/SumologicExtension`: removing collector name from credential path for sumologic extension (#42511)
- `opensearchexporter`: Add support for bodymap mapping mode (#41654)
  The bodymap mapping mode supports only logs and uses the body of a log record as the exact content of the OpenSearch document, without any transformation.
- `tailsamplingprocessor`: Add support for extensions that implement sampling policies. (#31582)
  Extension support for tailsamplingprocessor is still in development and the interfaces may change at any time.
  
- `telemetrygen`: Add span links support to telemetrygen (#43007)
  The new --span-links flag allows generating spans with links to previously created spans.
  Each span can link to random existing span contexts, creating relationships between spans for testing
  distributed tracing scenarios. Links include attributes for link type and index identification.
  
- `telemetrygen`: Add load size to telemetrygen metrics and logs. (#42322)

### 🧰 Bug fixes 🧰

- `awsxrayexporter`: infer downstream service for producer spans (#40995)
- `azureeventhubreceiver`: Use `$Default` as the default consumer group with the new azeventhubs SDK (#43049)
- `azureeventhubreceiver`: Offset configuration option is now correctly honored, and the default start position is set to latest. (#38487)
- `elasticsearchexporter`: Fix routing of collector self-telemetry data (#42679)
- `elasticsearchexporter`: profiling: fix fetching location for stack (#42891)
- `receiver/googlecloudmonitoring`: Add metric labels from Google Cloud metrics to all OTel metric attributes (#42232)
- `jmxreceiver`: Fix the jmx-scraper hash for version 1.49.0 (#121332)
- `postgreqsqlreceiver`: Fix for memory leak when using top queries (#43076)
- `ntpreceiver`: Fix missing resource attribute 'ntp.host' to ntpreceiver metrics (#43129)
- `receiver/k8seventsreceiver`: Prevent potential panic in the events receiver by safely checking that informer objects are *corev1.Event before handling them. (#43014)
- `awscloudwatchlogexporter, awsemfexporter, awsxrayexporter`: Fix support for role_arn (STS, short-lived token authentication). (#42115)
- `jmxreceiver`: restart the java process on error (#42138)
  Previously, the java process would not restart on error. By default, this receiver will now
  always restart the process on error.
  
- `processor/k8sattributes`: Use podUID instead podName to determine which pods should be deleted from cache (#42978)
- `kafka`: Fix support for protocol_version in franz-go client (#42795)
- `libhoneyreceiver`: return full array of statuses per event (#42272)
  Libhoney has a per-event-within-each-batch response code array for each batch received. This has now been implemented for both initial parsing errors as well as downstream consumer errors.
- `telemetrygen`: Publish int and bool attributes for logs (#43090)
- `oracledbreceiver`: Fix for wrong trace id in oracle top query records (#43111)
- `oracledbreceiver`: Fix for memory leak in top queries and query samples collection. (#43074)
- `prometheusexporter, prometheusremotewriteexporter`: Connect pkg.translator.prometheus.PermissiveLabelSanitization with relevant logic. (#43077)
- `postgresqlreceiver`: Properly set `network.peer.address` attribute (#42447)
- `postgresqlreceiver`: Fix for inflated metric values in query metrics collection (#43071)
- `prometheusexporter`: Fix 'failed to build namespace' logged as error when namespace is not configured (#43015)
- `signalfxexporter`: Add HostID resource attribute to Histogram data in OTLP format (#42905)
- `statsdreceiver`: Fix a data race in statsdreceiver on shutdown (#42878)

<!-- previous-version -->


## v1.43.0/v0.137.0

### 💡 Enhancements 💡

- `cmd/mdatagen`: Improve validation for resource attribute `enabled` field in metadata files (#12722)
  Resource attributes now require an explicit `enabled` field in metadata.yaml files, while regular attributes
  are prohibited from having this field. This improves validation and prevents configuration errors.
  
- `all`: Changelog entries will now have their component field checked against a list of valid components. (#13924)
  This will ensure a more standardized changelog format which makes it easier to parse.
- `pkg/pdata`: Mark featuregate pdata.useCustomProtoEncoding as stable (#13883)

<!-- previous-version -->
