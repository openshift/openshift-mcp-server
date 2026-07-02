package mustgather

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"sigs.k8s.io/yaml"
)

// archiveURIPrefix is the fixed prefix of every must-gather resource URI. The
// path segment immediately after it is the archive ID (mg-XXXX-YYYYYYYY).
const archiveURIPrefix = "must-gather://local/"

// initMCPResources returns no static resources: because every archive is
// addressed by ID, all resources are exposed as templates (see
// initMCPResourceTemplates).
func initMCPResources() []api.ServerResource {
	return nil
}

func initMCPResourceTemplates() []api.ServerResourceTemplate {
	return []api.ServerResourceTemplate{
		{
			ResourceTemplate: api.ResourceTemplate{
				URITemplate: "must-gather://local/{archive_id}",
				Name:        "must-gather",
				Description: "Loaded must-gather archive metadata. Use the must_gather_archive_id from mustgather_list as {archive_id}.",
				MIMEType:    "text/plain",
			},
			Handler: resourceCurrentArchive,
		},
		{
			ResourceTemplate: api.ResourceTemplate{
				URITemplate: "must-gather://local/{archive_id}/namespaces",
				Name:        "must-gather-namespaces",
				Description: "List of all namespaces in the must-gather archive",
				MIMEType:    "text/plain",
			},
			Handler: resourceNamespaces,
		},
		{
			ResourceTemplate: api.ResourceTemplate{
				URITemplate: "must-gather://local/{archive_id}/etcd/members",
				Name:        "must-gather-etcd-members",
				Description: "ETCD cluster member list from the must-gather archive",
				MIMEType:    "application/json",
			},
			Handler: resourceETCDMembers,
		},
		{
			ResourceTemplate: api.ResourceTemplate{
				URITemplate: "must-gather://local/{archive_id}/etcd/endpoint-status",
				Name:        "must-gather-etcd-endpoint-status",
				Description: "ETCD endpoint status from the must-gather archive",
				MIMEType:    "application/json",
			},
			Handler: resourceETCDEndpointStatus,
		},
		{
			ResourceTemplate: api.ResourceTemplate{
				URITemplate: "must-gather://local/{archive_id}/prometheus/config",
				Name:        "must-gather-prometheus-config",
				Description: "Prometheus configuration summary from the must-gather archive",
				MIMEType:    "text/plain",
			},
			Handler: resourcePrometheusConfig,
		},
		{
			ResourceTemplate: api.ResourceTemplate{
				URITemplate: "must-gather://local/{archive_id}/alertmanager/status",
				Name:        "must-gather-alertmanager-status",
				Description: "AlertManager status from the must-gather archive",
				MIMEType:    "text/plain",
			},
			Handler: resourceAlertManagerStatus,
		},
		{
			ResourceTemplate: api.ResourceTemplate{
				URITemplate: "must-gather://local/{archive_id}/resources/{group}/{version}/{kind}/{namespace}/{name}",
				Name:        "must-gather-resource",
				Description: "A specific Kubernetes resource from the must-gather archive as YAML. Use '-' for empty group (core API) or cluster-scoped namespace.",
				MIMEType:    "text/yaml",
			},
			Handler: resourceGet,
		},
	}
}

// archiveIDFromURI extracts the archive ID (the first path segment after the
// must-gather://local/ prefix) from a resource URI.
func archiveIDFromURI(uri string) (string, error) {
	if !strings.HasPrefix(uri, archiveURIPrefix) {
		return "", fmt.Errorf("invalid must-gather resource URI: %s", uri)
	}
	rest := strings.TrimPrefix(uri, archiveURIPrefix)
	id := rest
	if i := strings.Index(rest, "/"); i >= 0 {
		id = rest[:i]
	}
	if id == "" {
		return "", fmt.Errorf("must-gather resource URI missing archive ID: %s", uri)
	}
	return id, nil
}

// providerForURI resolves the provider for the archive addressed by uri.
func providerForURI(cfg api.MustGatherDirsProvider, uri string) (*mg.Provider, error) {
	id, err := archiveIDFromURI(uri)
	if err != nil {
		return nil, err
	}
	return providerForArchive(cfg, id)
}

func resourceCurrentArchive(_ context.Context, cfg api.BaseConfig, uri string) (*api.ResourceContent, error) {
	p, err := providerForURI(cfg, uri)
	if err != nil {
		return nil, err
	}
	metadata := p.GetMetadata()
	content := fmt.Sprintf("Must-Gather Archive\nPath: %s\nVersion: %s\nTimestamp: %s\nResources: %d\nNamespaces: %d\n",
		metadata.Path, metadata.Version, metadata.Timestamp,
		metadata.ResourceCount, metadata.NamespaceCount)
	return &api.ResourceContent{Text: content}, nil
}

func resourceNamespaces(_ context.Context, cfg api.BaseConfig, uri string) (*api.ResourceContent, error) {
	p, err := providerForURI(cfg, uri)
	if err != nil {
		return nil, err
	}
	namespaces := p.ListNamespaces()
	sort.Strings(namespaces)
	output := fmt.Sprintf("Found %d namespaces:\n\n", len(namespaces))
	output += strings.Join(namespaces, "\n") + "\n"
	return &api.ResourceContent{Text: output}, nil
}

func resourceETCDMembers(_ context.Context, cfg api.BaseConfig, uri string) (*api.ResourceContent, error) {
	p, err := providerForURI(cfg, uri)
	if err != nil {
		return nil, err
	}
	data, err := p.ReadETCDFile("member_list.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read ETCD member list: %w", err)
	}
	return &api.ResourceContent{Text: string(data)}, nil
}

func resourceETCDEndpointStatus(_ context.Context, cfg api.BaseConfig, uri string) (*api.ResourceContent, error) {
	p, err := providerForURI(cfg, uri)
	if err != nil {
		return nil, err
	}
	data, err := p.ReadETCDFile("endpoint_status.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read ETCD endpoint status: %w", err)
	}
	return &api.ResourceContent{Text: string(data)}, nil
}

func resourcePrometheusConfig(_ context.Context, cfg api.BaseConfig, uri string) (*api.ResourceContent, error) {
	p, err := providerForURI(cfg, uri)
	if err != nil {
		return nil, err
	}

	output := "## Prometheus Configuration Summary\n\n"

	config, err := p.GetPrometheusConfig()
	if err != nil {
		output += fmt.Sprintf("Config not available: %v\n\n", err)
	} else {
		configYAML := config.YAML
		lines := strings.Split(configYAML, "\n")
		if len(lines) > 100 {
			output += fmt.Sprintf("Configuration (%d lines, showing first 100):\n\n", len(lines))
			output += strings.Join(lines[:100], "\n") + "\n...\n\n"
		} else {
			output += "Configuration:\n\n" + configYAML + "\n\n"
		}
	}

	flags, err := p.GetPrometheusFlags()
	if err != nil {
		output += fmt.Sprintf("Flags not available: %v\n\n", err)
	} else {
		output += "### Key Flags\n\n"
		keyFlags := []string{
			"storage.tsdb.retention.time",
			"storage.tsdb.retention.size",
			"storage.tsdb.path",
			"web.listen-address",
			"web.external-url",
			"rules.alert.for-outage-tolerance",
			"rules.alert.for-grace-period",
		}
		for _, key := range keyFlags {
			if val, ok := flags[key]; ok {
				output += fmt.Sprintf("%-40s %s\n", key, val)
			}
		}
		output += "\n"
	}

	return &api.ResourceContent{Text: output}, nil
}

func resourceAlertManagerStatus(_ context.Context, cfg api.BaseConfig, uri string) (*api.ResourceContent, error) {
	p, err := providerForURI(cfg, uri)
	if err != nil {
		return nil, err
	}

	status, err := p.GetAlertManagerStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get AlertManager status: %w", err)
	}

	output := "## AlertManager Status\n\n"
	output += fmt.Sprintf("Cluster Status: %s\n", healthSymbol(status.Cluster.Status))
	output += fmt.Sprintf("Uptime: %s\n\n", status.Uptime)

	output += "### Version\n\n"
	output += fmt.Sprintf("Version:    %s\n", status.VersionInfo.Version)
	output += fmt.Sprintf("Revision:   %s\n", status.VersionInfo.Revision)
	output += fmt.Sprintf("Branch:     %s\n", status.VersionInfo.Branch)
	output += fmt.Sprintf("Build Date: %s\n", status.VersionInfo.BuildDate)
	output += fmt.Sprintf("Go Version: %s\n\n", status.VersionInfo.GoVersion)

	if len(status.Cluster.Peers) > 0 {
		output += "### Cluster Peers\n\n"
		for _, peer := range status.Cluster.Peers {
			output += fmt.Sprintf("- %s (%s)\n", peer.Name, peer.Address)
		}
		output += "\n"
	}

	return &api.ResourceContent{Text: output}, nil
}

func resourceGet(_ context.Context, cfg api.BaseConfig, uri string) (*api.ResourceContent, error) {
	id, err := archiveIDFromURI(uri)
	if err != nil {
		return nil, err
	}
	p, err := providerForArchive(cfg, id)
	if err != nil {
		return nil, err
	}

	// Parse URI: must-gather://local/{archive_id}/resources/{group}/{version}/{kind}/{namespace}/{name}
	prefix := archiveURIPrefix + id + "/resources/"
	if !strings.HasPrefix(uri, prefix) {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}
	parts := strings.SplitN(strings.TrimPrefix(uri, prefix), "/", 5)
	if len(parts) != 5 {
		return nil, fmt.Errorf("resource URI must have format: must-gather://local/{archive_id}/resources/{group}/{version}/{kind}/{namespace}/{name}")
	}
	group, version, kind, namespace, name := parts[0], parts[1], parts[2], parts[3], parts[4]

	// "-" represents empty group (core API) or cluster-scoped resources (no namespace)
	if group == "-" {
		group = ""
	}
	if namespace == "-" {
		namespace = ""
	}

	gvk := parseGVK(apiVersionFromGroupVersion(group, version), kind)
	obj := p.GetResource(gvk, name, namespace)
	if obj == nil {
		return nil, fmt.Errorf("resource %s/%s not found", kind, name)
	}

	yamlBytes, err := yaml.Marshal(obj.Object)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource: %w", err)
	}

	return &api.ResourceContent{Text: string(yamlBytes)}, nil
}

func apiVersionFromGroupVersion(group, version string) string {
	if group == "" {
		return version
	}
	return group + "/" + version
}
