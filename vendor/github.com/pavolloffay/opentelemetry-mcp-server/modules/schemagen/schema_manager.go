package schemagen

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/philippgille/chromem-go"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// ComponentType represents the type of OpenTelemetry component
type ComponentType string

const (
	ComponentTypeReceiver  ComponentType = "receiver"
	ComponentTypeProcessor ComponentType = "processor"
	ComponentTypeExporter  ComponentType = "exporter"
	ComponentTypeExtension ComponentType = "extension"
	ComponentTypeConnector ComponentType = "connector"
)

// ComponentSchema represents a YAML schema for an OpenTelemetry component
type ComponentSchema struct {
	Name    string                 `json:"name"`
	Type    ComponentType          `json:"type"`
	Version string                 `json:"version,omitempty"`
	Schema  map[string]interface{} `json:"schema"`
}

// DeprecatedField represents a deprecated field with its information
type DeprecatedField struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// DocumentSearchResult represents a search result from the RAG database
type DocumentSearchResult struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Metadata   map[string]string `json:"metadata"`
	Similarity float32           `json:"similarity"`
	Component  string            `json:"component,omitempty"`
	Version    string            `json:"version,omitempty"`
	FilePath   string            `json:"file_path,omitempty"`
}

// SchemaManager manages component schemas and documentation RAG database
type SchemaManager struct {
	cache          map[string]*ComponentSchema
	schemaFS       fs.FS
	schemaBasePath string
	ragDB          *chromem.DB
	ragCollection  *chromem.Collection
	ragMutex       sync.RWMutex
	ragInit        sync.Once
}

// NewSchemaManagerFromDir creates a new schema manager using schemas from the specified directory.
// The directory should contain version subdirectories (e.g., "0.139.0/") with schema files.
func NewSchemaManagerFromDir(schemaDir string) (*SchemaManager, error) {
	info, err := os.Stat(schemaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to access schema directory %s: %w", schemaDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("schema path %s is not a directory", schemaDir)
	}

	return &SchemaManager{
		cache:          make(map[string]*ComponentSchema),
		schemaFS:       os.DirFS(schemaDir),
		schemaBasePath: ".",
	}, nil
}

// NewSchemaManagerFromFS creates a new schema manager using schemas from the provided filesystem.
// This allows using an embed.FS or any other fs.FS implementation.
// The basePath should be the path within the filesystem where version subdirectories are located
// (e.g., "schemas" if the structure is schemas/0.139.0/receivers/...).
func NewSchemaManagerFromFS(filesystem fs.FS, basePath string) *SchemaManager {
	return &SchemaManager{
		cache:          make(map[string]*ComponentSchema),
		schemaFS:       filesystem,
		schemaBasePath: basePath,
	}
}

// versionPath returns the filesystem path for a specific version
func (sm *SchemaManager) versionPath(version string) string {
	if sm.schemaBasePath == "." {
		return version
	}
	return filepath.Join(sm.schemaBasePath, version)
}

// createSimpleEmbeddingFunc creates a simple hash-based embedding function for testing
func createSimpleEmbeddingFunc() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		h1 := fnv.New64a()
		h2 := fnv.New64()
		h1.Write([]byte(text))
		h2.Write([]byte(text))

		hash1 := h1.Sum64()
		hash2 := h2.Sum64()

		md5Hash := md5.Sum([]byte(text))

		embedding := make([]float32, 384)

		for i := 0; i < 384; i++ {
			var value uint64
			if i < 128 {
				value = hash1 + uint64(i)
			} else if i < 256 {
				value = hash2 + uint64(i)
			} else {
				byteIdx := (i - 256) % 16
				value = uint64(md5Hash[byteIdx]) + uint64(i)
			}

			embedding[i] = float32(int32(value)) / float32(math.MaxInt32)
		}

		var norm float32
		for _, val := range embedding {
			norm += val * val
		}
		norm = float32(math.Sqrt(float64(norm)))

		if norm > 0 {
			for i := range embedding {
				embedding[i] /= norm
			}
		}

		return embedding, nil
	}
}

// initRAGDatabase initializes the RAG database and indexes all markdown files
func (sm *SchemaManager) initRAGDatabase() error {
	var err error
	sm.ragInit.Do(func() {
		sm.ragDB = chromem.NewDB()

		embeddingFunc := createSimpleEmbeddingFunc()
		metadata := map[string]string{
			"description": "OpenTelemetry Collector Component Documentation",
		}

		collection, collErr := sm.ragDB.CreateCollection("otel-docs", metadata, embeddingFunc)
		if collErr != nil {
			err = fmt.Errorf("failed to create RAG collection: %w", collErr)
			return
		}
		sm.ragCollection = collection

		versions, vErr := sm.GetAllVersions()
		if vErr != nil {
			err = fmt.Errorf("failed to get versions for RAG indexing: %w", vErr)
			return
		}

		for _, version := range versions {
			if indexErr := sm.indexMarkdownFiles(version); indexErr != nil {
				err = fmt.Errorf("failed to index markdown files for version %s: %w", version, indexErr)
				return
			}
		}
	})
	return err
}

// indexMarkdownFiles indexes all markdown files for a specific version
func (sm *SchemaManager) indexMarkdownFiles(version string) error {
	schemaPath := sm.versionPath(version)
	entries, err := fs.ReadDir(sm.schemaFS, schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema directory for version %s: %w", version, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(schemaPath, entry.Name())
		content, err := fs.ReadFile(sm.schemaFS, filePath)
		if err != nil {
			fmt.Printf("Warning: failed to read markdown file %s: %v\n", filePath, err)
			continue
		}

		componentName := strings.TrimSuffix(entry.Name(), ".md")
		metadata := map[string]string{
			"version":   version,
			"component": componentName,
			"file_path": filePath,
			"file_type": "markdown",
		}

		parts := strings.SplitN(componentName, "_", 2)
		if len(parts) == 2 {
			metadata["component_type"] = parts[0]
			metadata["component_name"] = parts[1]
		}

		docID := fmt.Sprintf("%s/%s", version, componentName)
		doc := chromem.Document{
			ID:       docID,
			Content:  string(content),
			Metadata: metadata,
		}

		if err := sm.ragCollection.AddDocument(context.Background(), doc); err != nil {
			fmt.Printf("Warning: failed to add document %s to RAG database: %v\n", docID, err)
			continue
		}
	}

	return nil
}

// GetComponentSchema returns the YAML schema for a specific component
func (sm *SchemaManager) GetComponentSchema(componentType ComponentType, componentName string, version string) (*ComponentSchema, error) {
	cacheKey := fmt.Sprintf("%s_%s_%s", componentType, componentName, version)

	if schema, exists := sm.cache[cacheKey]; exists {
		return schema, nil
	}

	schema, err := sm.loadSchemaFromFile(componentType, componentName, version)
	if err != nil {
		return nil, err
	}

	sm.cache[cacheKey] = schema

	return schema, nil
}

// GetComponentSchemaJSON returns the YAML schema as a JSON byte array
func (sm *SchemaManager) GetComponentSchemaJSON(componentType ComponentType, componentName string, version string) ([]byte, error) {
	schema, err := sm.GetComponentSchema(componentType, componentName, version)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(schema.Schema, "", "  ")
}

// ListAvailableComponents returns a list of all available components by type
func (sm *SchemaManager) ListAvailableComponents(version string) (map[ComponentType][]string, error) {
	return sm.listComponents(version)
}

// ValidateComponentJSON validates a component configuration JSON against its schema
func (sm *SchemaManager) ValidateComponentJSON(componentType ComponentType, componentName string, version string, jsonData []byte) (*gojsonschema.Result, error) {
	componentSchema, err := sm.GetComponentSchema(componentType, componentName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for %s %s v%s: %w", componentType, componentName, version, err)
	}

	schemaBytes, err := json.Marshal(componentSchema.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema for %s %s: %w", componentType, componentName, err)
	}

	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validation failed for %s %s: %w", componentType, componentName, err)
	}

	return result, nil
}

// ValidateComponentYAML validates a component configuration YAML against its schema
func (sm *SchemaManager) ValidateComponentYAML(componentType ComponentType, componentName string, version string, yamlData []byte) (*gojsonschema.Result, error) {
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML data: %w", err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON for validation: %w", err)
	}

	return sm.ValidateComponentJSON(componentType, componentName, version, jsonData)
}

// GetComponentReadme returns the README content for a specific component
func (sm *SchemaManager) GetComponentReadme(componentType ComponentType, componentName string, version string) (string, error) {
	filename := fmt.Sprintf("%s_%s.md", componentType, componentName)

	schemaPath := sm.versionPath(version)
	filePath := filepath.Join(schemaPath, filename)
	data, err := fs.ReadFile(sm.schemaFS, filePath)
	if err != nil {
		return "", fmt.Errorf("README not found for component %s %s v%s", componentType, componentName, version)
	}

	return string(data), nil
}

// GetChangelog returns the changelog content for a specific collector version
func (sm *SchemaManager) GetChangelog(version string) (string, error) {
	schemaPath := sm.versionPath(version)
	filePath := filepath.Join(schemaPath, "changelog.md")
	data, err := fs.ReadFile(sm.schemaFS, filePath)
	if err != nil {
		return "", fmt.Errorf("changelog not found for version %s", version)
	}

	return string(data), nil
}

// listComponents lists components from the schema filesystem
func (sm *SchemaManager) listComponents(version string) (map[ComponentType][]string, error) {
	components := make(map[ComponentType][]string)

	schemaPath := sm.versionPath(version)
	entries, err := fs.ReadDir(sm.schemaFS, schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")

		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			continue
		}

		componentType := ComponentType(parts[0])
		componentName := parts[1]

		if !IsValidComponentType(componentType) {
			continue
		}

		components[componentType] = append(components[componentType], componentName)
	}

	return components, nil
}

// loadSchemaFromFile loads a schema from the schema filesystem
func (sm *SchemaManager) loadSchemaFromFile(componentType ComponentType, componentName string, version string) (*ComponentSchema, error) {
	filename := fmt.Sprintf("%s_%s.yaml", componentType, componentName)

	schemaPath := sm.versionPath(version)
	filePath := filepath.Join(schemaPath, filename)
	data, err := fs.ReadFile(sm.schemaFS, filePath)
	if err != nil {
		return nil, fmt.Errorf("schema not found for component %s %s", componentType, componentName)
	}

	var schemaData map[string]interface{}
	if err := yaml.Unmarshal(data, &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse schema YAML for %s %s: %w", componentType, componentName, err)
	}

	componentVersion := version

	return &ComponentSchema{
		Name:    componentName,
		Type:    componentType,
		Version: componentVersion,
		Schema:  schemaData,
	}, nil
}

// IsValidComponentType checks if the component type is valid
func IsValidComponentType(componentType ComponentType) bool {
	switch componentType {
	case ComponentTypeReceiver, ComponentTypeProcessor, ComponentTypeExporter, ComponentTypeExtension, ComponentTypeConnector:
		return true
	default:
		return false
	}
}

// GetLatestVersion returns the latest version available in the schemas directory
func (sm *SchemaManager) GetLatestVersion() (string, error) {
	entries, err := fs.ReadDir(sm.schemaFS, sm.schemaBasePath)
	if err != nil {
		return "", fmt.Errorf("failed to read schemas directory: %w", err)
	}

	var latestVersion string
	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			if strings.Contains(version, ".") {
				if latestVersion == "" || version > latestVersion {
					latestVersion = version
				}
			}
		}
	}

	if latestVersion == "" {
		return "", fmt.Errorf("no versions found in schemas directory")
	}

	return latestVersion, nil
}

// GetAllVersions returns all versions available in the schemas directory
func (sm *SchemaManager) GetAllVersions() ([]string, error) {
	entries, err := fs.ReadDir(sm.schemaFS, sm.schemaBasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schemas directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			if strings.Contains(version, ".") {
				versions = append(versions, version)
			}
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found in schemas directory")
	}

	return versions, nil
}

// GetComponentNames returns all component names for a given version and component type
func (sm *SchemaManager) GetComponentNames(componentType ComponentType, version string) ([]string, error) {
	if !IsValidComponentType(componentType) {
		return nil, fmt.Errorf("invalid component type: %s", componentType)
	}

	schemaPath := sm.versionPath(version)
	entries, err := fs.ReadDir(sm.schemaFS, schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema directory for version %s: %w", version, err)
	}

	var componentNames []string
	prefix := string(componentType) + "_"

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		if strings.HasPrefix(entry.Name(), prefix) {
			name := strings.TrimSuffix(entry.Name(), ".yaml")
			componentName := strings.TrimPrefix(name, prefix)
			if componentName != "" {
				componentNames = append(componentNames, componentName)
			}
		}
	}

	if len(componentNames) == 0 {
		return nil, fmt.Errorf("no %s components found for version %s", componentType, version)
	}

	return componentNames, nil
}

// QueryDocumentation searches the RAG database for relevant documentation based on the query text for a specific version
func (sm *SchemaManager) QueryDocumentation(query string, version string, maxResults int) ([]DocumentSearchResult, error) {
	sm.ragMutex.RLock()
	defer sm.ragMutex.RUnlock()

	if err := sm.initRAGDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize RAG database: %w", err)
	}

	where := map[string]string{
		"version": version,
	}

	results, err := sm.ragCollection.Query(context.Background(), query, maxResults, where, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query RAG database: %w", err)
	}

	searchResults := make([]DocumentSearchResult, len(results))
	for i, result := range results {
		searchResult := DocumentSearchResult{
			ID:         result.ID,
			Content:    result.Content,
			Metadata:   result.Metadata,
			Similarity: result.Similarity,
		}

		if component, exists := result.Metadata["component"]; exists {
			searchResult.Component = component
		}
		if resultVersion, exists := result.Metadata["version"]; exists {
			searchResult.Version = resultVersion
		}
		if filePath, exists := result.Metadata["file_path"]; exists {
			searchResult.FilePath = filePath
		}

		searchResults[i] = searchResult
	}

	return searchResults, nil
}

// QueryDocumentationWithFilters searches the RAG database with additional filtering options beyond version.
func (sm *SchemaManager) QueryDocumentationWithFilters(query string, maxResults int, componentType, componentName, version string) ([]DocumentSearchResult, error) {
	sm.ragMutex.RLock()
	defer sm.ragMutex.RUnlock()

	if err := sm.initRAGDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize RAG database: %w", err)
	}

	where := make(map[string]string)
	if componentType != "" {
		where["component_type"] = componentType
	}
	if componentName != "" {
		where["component_name"] = componentName
	}
	if version != "" {
		where["version"] = version
	}

	results, err := sm.ragCollection.Query(context.Background(), query, maxResults, where, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query RAG database with filters: %w", err)
	}

	searchResults := make([]DocumentSearchResult, len(results))
	for i, result := range results {
		searchResult := DocumentSearchResult{
			ID:         result.ID,
			Content:    result.Content,
			Metadata:   result.Metadata,
			Similarity: result.Similarity,
		}

		if component, exists := result.Metadata["component"]; exists {
			searchResult.Component = component
		}
		if resultVersion, exists := result.Metadata["version"]; exists {
			searchResult.Version = resultVersion
		}
		if filePath, exists := result.Metadata["file_path"]; exists {
			searchResult.FilePath = filePath
		}

		searchResults[i] = searchResult
	}

	return searchResults, nil
}

// GetDeprecatedFields returns a list of deprecated fields with their information for a specific component
func (sm *SchemaManager) GetDeprecatedFields(componentType ComponentType, componentName string, version string) ([]DeprecatedField, error) {
	schema, err := sm.GetComponentSchema(componentType, componentName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for %s %s v%s: %w", componentType, componentName, version, err)
	}

	var deprecatedFields []DeprecatedField

	sm.findDeprecatedFields(schema.Schema, "", &deprecatedFields)

	return deprecatedFields, nil
}

// findDeprecatedFields recursively searches for deprecated fields in a JSON schema
func (sm *SchemaManager) findDeprecatedFields(schema map[string]interface{}, currentPath string, deprecatedFields *[]DeprecatedField) {
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for fieldName, fieldSchema := range properties {
			var fieldPath string
			if currentPath == "" {
				fieldPath = fieldName
			} else {
				fieldPath = currentPath + "." + fieldName
			}

			if fieldSchemaMap, ok := fieldSchema.(map[string]interface{}); ok {
				if deprecated, exists := fieldSchemaMap["deprecated"]; exists {
					if deprecatedBool, ok := deprecated.(bool); ok && deprecatedBool {
						description := ""
						if desc, exists := fieldSchemaMap["description"]; exists {
							if descStr, ok := desc.(string); ok {
								description = descStr
							}
						}

						fieldType := ""
						if fType, exists := fieldSchemaMap["type"]; exists {
							if typeStr, ok := fType.(string); ok {
								fieldType = typeStr
							}
						}

						deprecatedField := DeprecatedField{
							Name:        fieldPath,
							Description: description,
							Type:        fieldType,
						}

						*deprecatedFields = append(*deprecatedFields, deprecatedField)
					}
				}

				sm.findDeprecatedFields(fieldSchemaMap, fieldPath, deprecatedFields)
			}
		}
	}
}
