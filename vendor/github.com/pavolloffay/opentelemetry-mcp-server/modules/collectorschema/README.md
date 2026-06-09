# OpenTelemetry collector config schema

This project creates JSON schema library for the OpenTelemetry collector.

Each collector component has its own JSON schema. The schemas are also versioned.

## How does it work?

This library uses the [OpenTelemetry collector builder (OCB)](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder).
OCB generates Golang code from the supplied [manifest.yaml](manifest-0.138.0.yaml) and this library creates a JSON schema for all collector components.
Alongside the JSON schema there is also a readme file for each component.

## How to use it?

```go

import "github.com/pavolloffay/opentelemetry-mcp-server/modules/collectorschema"

schemaManager := collectorschema.NewSchemaManager()

readme, err := schemaManager.GetComponentReadme(collectorschema.ComponentType(componentType), componentName, version)
schemaJSON, err := schemaManager.GetComponentSchemaJSON(collectorschema.ComponentType(componentType), componentName, version)
validationResult, err := schemaManager.ValidateComponentJSON(collectorschema.ComponentType(componentType), componentName, version, []byte(config))
```