package collectorschema

import (
	"context"
	"crypto/md5"
	"embed"
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

//go:embed schemas
var embeddedSchemas embed.FS

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

// NewSchemaManager creates a new schema manager using embedded schemas
func NewSchemaManager() *SchemaManager {
	return &SchemaManager{
		cache:          make(map[string]*ComponentSchema),
		schemaFS:       embeddedSchemas,
		schemaBasePath: "schemas",
	}
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
// This avoids external API dependencies and creates deterministic embeddings
func createSimpleEmbeddingFunc() chromem.EmbeddingFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		// Create a simple embedding using text hashes
		// This is for testing purposes only and not suitable for production

		// Use multiple hash functions to create a 384-dimensional embedding
		h1 := fnv.New64a()
		h2 := fnv.New64()
		h1.Write([]byte(text))
		h2.Write([]byte(text))

		hash1 := h1.Sum64()
		hash2 := h2.Sum64()

		// Create MD5 hash for additional entropy
		md5Hash := md5.Sum([]byte(text))

		embedding := make([]float32, 384) // Standard embedding dimension

		// Fill embedding with normalized values derived from hashes
		for i := 0; i < 384; i++ {
			var value uint64
			if i < 128 {
				value = hash1 + uint64(i)
			} else if i < 256 {
				value = hash2 + uint64(i)
			} else {
				// Use MD5 bytes for remaining dimensions
				byteIdx := (i - 256) % 16
				value = uint64(md5Hash[byteIdx]) + uint64(i)
			}

			// Convert to float and normalize to [-1, 1]
			embedding[i] = float32(int32(value)) / float32(math.MaxInt32)
		}

		// Normalize the embedding vector
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
		// Create a new ChromaDB instance
		sm.ragDB = chromem.NewDB()

		// Create a collection for documentation
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

		// Get all versions to index documentation from all versions
		versions, vErr := sm.GetAllVersions()
		if vErr != nil {
			err = fmt.Errorf("failed to get versions for RAG indexing: %w", vErr)
			return
		}

		// Index all markdown files across all versions
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

		// Read the markdown file
		filePath := filepath.Join(schemaPath, entry.Name())
		content, err := fs.ReadFile(sm.schemaFS, filePath)
		if err != nil {
			// Log warning but continue with other files
			fmt.Printf("Warning: failed to read markdown file %s: %v\n", filePath, err)
			continue
		}

		// Create document metadata
		componentName := strings.TrimSuffix(entry.Name(), ".md")
		metadata := map[string]string{
			"version":    version,
			"component":  componentName,
			"file_path":  filePath,
			"file_type":  "markdown",
		}

		// Parse component type and name
		parts := strings.SplitN(componentName, "_", 2)
		if len(parts) == 2 {
			metadata["component_type"] = parts[0]
			metadata["component_name"] = parts[1]
		}

		// Create document for RAG database
		docID := fmt.Sprintf("%s/%s", version, componentName)
		doc := chromem.Document{
			ID:       docID,
			Content:  string(content),
			Metadata: metadata,
		}

		// Add document to RAG collection
		if err := sm.ragCollection.AddDocument(context.Background(), doc); err != nil {
			// Log warning but continue with other files
			fmt.Printf("Warning: failed to add document %s to RAG database: %v\n", docID, err)
			continue
		}
	}

	return nil
}

// GetComponentSchema returns the YAML schema for a specific component
func (sm *SchemaManager) GetComponentSchema(componentType ComponentType, componentName string, version string) (*ComponentSchema, error) {
	// Create cache key
	cacheKey := fmt.Sprintf("%s_%s_%s", componentType, componentName, version)

	// Check cache first
	if schema, exists := sm.cache[cacheKey]; exists {
		return schema, nil
	}

	// Load schema from file
	schema, err := sm.loadSchemaFromFile(componentType, componentName, version)
	if err != nil {
		return nil, err
	}

	// Cache the result
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
	// Get the component schema
	componentSchema, err := sm.GetComponentSchema(componentType, componentName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for %s %s v%s: %w", componentType, componentName, version, err)
	}

	// Convert schema to JSON bytes for gojsonschema
	schemaBytes, err := json.Marshal(componentSchema.Schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema for %s %s: %w", componentType, componentName, err)
	}

	// Create schema loader
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	// Create document loader from the provided JSON data
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	// Validate the document against the schema
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validation failed for %s %s: %w", componentType, componentName, err)
	}

	return result, nil
}

// ValidateComponentYAML validates a component configuration YAML against its schema
func (sm *SchemaManager) ValidateComponentYAML(componentType ComponentType, componentName string, version string, yamlData []byte) (*gojsonschema.Result, error) {
	// Parse YAML data to interface{}
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML data: %w", err)
	}

	// Convert to JSON for validation
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to JSON for validation: %w", err)
	}

	// Use existing JSON validation function
	return sm.ValidateComponentJSON(componentType, componentName, version, jsonData)
}

// GetComponentReadme returns the README content for a specific component
func (sm *SchemaManager) GetComponentReadme(componentType ComponentType, componentName string, version string) (string, error) {
	// Construct filename (format: type_name.md)
	filename := fmt.Sprintf("%s_%s.md", componentType, componentName)

	// Load from filesystem
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
	// Load changelog.md from filesystem
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

	// Read schema directory
	schemaPath := sm.versionPath(version)
	entries, err := fs.ReadDir(sm.schemaFS, schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Remove .yaml extension
		name := strings.TrimSuffix(entry.Name(), ".yaml")

		// Parse component type and name from filename (format: type_name.yaml)
		parts := strings.SplitN(name, "_", 2)
		if len(parts) != 2 {
			continue // Skip files that don't match the expected format
		}

		componentType := ComponentType(parts[0])
		componentName := parts[1]

		// Validate component type
		if !isValidComponentType(componentType) {
			continue
		}

		components[componentType] = append(components[componentType], componentName)
	}

	return components, nil
}

// loadSchemaFromFile loads a schema from the schema filesystem
func (sm *SchemaManager) loadSchemaFromFile(componentType ComponentType, componentName string, version string) (*ComponentSchema, error) {
	// Construct filename (format: type_name.yaml)
	filename := fmt.Sprintf("%s_%s.yaml", componentType, componentName)

	// Load from filesystem
	schemaPath := sm.versionPath(version)
	filePath := filepath.Join(schemaPath, filename)
	data, err := fs.ReadFile(sm.schemaFS, filePath)
	if err != nil {
		return nil, fmt.Errorf("schema not found for component %s %s", componentType, componentName)
	}

	// Parse YAML schema
	var schemaData map[string]interface{}
	if err := yaml.Unmarshal(data, &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse schema YAML for %s %s: %w", componentType, componentName, err)
	}

	// Use the provided version
	componentVersion := version

	return &ComponentSchema{
		Name:    componentName,
		Type:    componentType,
		Version: componentVersion,
		Schema:  schemaData,
	}, nil
}

// isValidComponentType checks if the component type is valid
func isValidComponentType(componentType ComponentType) bool {
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
			// Check if the directory name looks like a version (contains dots)
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
			// Check if the directory name looks like a version (contains dots)
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
	// Validate component type
	if !isValidComponentType(componentType) {
		return nil, fmt.Errorf("invalid component type: %s", componentType)
	}

	// Read schema directory for the specific version
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

		// Check if the file matches the component type pattern (e.g., "receiver_otlp.yaml")
		if strings.HasPrefix(entry.Name(), prefix) {
			// Extract component name by removing prefix and .yaml suffix
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

// DocumentSearchResult represents a search result from the RAG database
type DocumentSearchResult struct {
	ID          string            `json:"id"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata"`
	Similarity  float32           `json:"similarity"`
	Component   string            `json:"component,omitempty"`
	Version     string            `json:"version,omitempty"`
	FilePath    string            `json:"file_path,omitempty"`
}

// QueryDocumentation searches the RAG database for relevant documentation based on the query text for a specific version
func (sm *SchemaManager) QueryDocumentation(query string, version string, maxResults int) ([]DocumentSearchResult, error) {
	sm.ragMutex.RLock()
	defer sm.ragMutex.RUnlock()

	// Initialize RAG database if not already done
	if err := sm.initRAGDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize RAG database: %w", err)
	}

	// Build where filter to restrict search to the specified version
	where := map[string]string{
		"version": version,
	}

	// Perform the search with version filter
	results, err := sm.ragCollection.Query(context.Background(), query, maxResults, where, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query RAG database: %w", err)
	}

	// Convert chromem results to our result structure
	searchResults := make([]DocumentSearchResult, len(results))
	for i, result := range results {
		searchResult := DocumentSearchResult{
			ID:         result.ID,
			Content:    result.Content,
			Metadata:   result.Metadata,
			Similarity: result.Similarity,
		}

		// Extract commonly used metadata fields for easier access
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
// Use this method when you need to filter by component type, component name, or version.
// For simple version-scoped searches, use QueryDocumentation instead.
func (sm *SchemaManager) QueryDocumentationWithFilters(query string, maxResults int, componentType, componentName, version string) ([]DocumentSearchResult, error) {
	sm.ragMutex.RLock()
	defer sm.ragMutex.RUnlock()

	// Initialize RAG database if not already done
	if err := sm.initRAGDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize RAG database: %w", err)
	}

	// Build where filter
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

	// Perform the search with filters
	results, err := sm.ragCollection.Query(context.Background(), query, maxResults, where, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query RAG database with filters: %w", err)
	}

	// Convert chromem results to our result structure
	searchResults := make([]DocumentSearchResult, len(results))
	for i, result := range results {
		searchResult := DocumentSearchResult{
			ID:         result.ID,
			Content:    result.Content,
			Metadata:   result.Metadata,
			Similarity: result.Similarity,
		}

		// Extract commonly used metadata fields for easier access
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
	// Get the component schema
	schema, err := sm.GetComponentSchema(componentType, componentName, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for %s %s v%s: %w", componentType, componentName, version, err)
	}

	var deprecatedFields []DeprecatedField

	// Recursively traverse the schema to find deprecated fields
	sm.findDeprecatedFields(schema.Schema, "", &deprecatedFields)

	return deprecatedFields, nil
}

// findDeprecatedFields recursively searches for deprecated fields in a JSON schema
func (sm *SchemaManager) findDeprecatedFields(schema map[string]interface{}, currentPath string, deprecatedFields *[]DeprecatedField) {
	// Check if this schema has properties
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		// Iterate through all properties
		for fieldName, fieldSchema := range properties {
			// Build the current field path
			var fieldPath string
			if currentPath == "" {
				fieldPath = fieldName
			} else {
				fieldPath = currentPath + "." + fieldName
			}

			// Check if the field schema is a map
			if fieldSchemaMap, ok := fieldSchema.(map[string]interface{}); ok {
				// Check if this field is marked as deprecated
				if deprecated, exists := fieldSchemaMap["deprecated"]; exists {
					if deprecatedBool, ok := deprecated.(bool); ok && deprecatedBool {
						// Extract field information
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

				// Recursively check nested objects
				sm.findDeprecatedFields(fieldSchemaMap, fieldPath, deprecatedFields)
			}
		}
	}
}
