package mustgather

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetPrometheusReplicaPath builds path to Prometheus replica data
func (p *Provider) GetPrometheusReplicaPath(replicaNum int) string {
	return filepath.Join(p.metadata.ContainerDir, "monitoring", "prometheus",
		fmt.Sprintf("prometheus-k8s-%d", replicaNum))
}

// GetPrometheusCommonPath builds path to common Prometheus data
func (p *Provider) GetPrometheusCommonPath() string {
	return filepath.Join(p.metadata.ContainerDir, "monitoring", "prometheus")
}

// GetAlertManagerPath builds path to AlertManager data
func (p *Provider) GetAlertManagerPath() string {
	return filepath.Join(p.metadata.ContainerDir, "monitoring", "alertmanager")
}

// ReadPrometheusJSON reads and parses a JSON file from a Prometheus replica directory
func (p *Provider) ReadPrometheusJSON(replicaPath, filename string, v any) error {
	dataFile := filepath.Join(replicaPath, filename)
	return ReadJSON(dataFile, v)
}

// GetPrometheusTSDB reads TSDB status for a replica
func (p *Provider) GetPrometheusTSDB(replicaNum int) (*TSDBStatus, error) {
	replicaPath := p.GetPrometheusReplicaPath(replicaNum)
	var resp TSDBStatusResponse
	if err := p.ReadPrometheusJSON(replicaPath, "status/tsdb.json", &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetPrometheusRuntimeInfo reads runtime info for a replica
func (p *Provider) GetPrometheusRuntimeInfo(replicaNum int) (*RuntimeInfo, error) {
	replicaPath := p.GetPrometheusReplicaPath(replicaNum)
	var resp RuntimeInfoResponse
	if err := p.ReadPrometheusJSON(replicaPath, "status/runtimeinfo.json", &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetPrometheusActiveTargets reads active targets for a replica
func (p *Provider) GetPrometheusActiveTargets(replicaNum int) ([]ActiveTarget, error) {
	replicaPath := p.GetPrometheusReplicaPath(replicaNum)
	var resp ActiveTargetsAPIResponse
	if err := p.ReadPrometheusJSON(replicaPath, "active-targets.json", &resp); err != nil {
		return nil, err
	}
	return resp.Data.ActiveTargets, nil
}

// GetPrometheusRules reads Prometheus rules from the common directory
func (p *Provider) GetPrometheusRules() (*RuleGroupsResponse, error) {
	promPath := p.GetPrometheusCommonPath()
	rulesFile := filepath.Join(promPath, "rules.json")
	var resp RuleGroupsAPIResponse
	if err := ReadJSON(rulesFile, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetAlertManagerStatus reads AlertManager status
func (p *Provider) GetAlertManagerStatus() (*AlertManagerStatus, error) {
	amPath := p.GetAlertManagerPath()
	statusFile := filepath.Join(amPath, "status.json")
	var status AlertManagerStatus
	if err := ReadJSON(statusFile, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// GetPrometheusConfig reads Prometheus config from the common directory
func (p *Provider) GetPrometheusConfig() (*ConfigResponse, error) {
	promPath := p.GetPrometheusCommonPath()
	configFile := filepath.Join(promPath, "status", "config.json")
	var config ConfigResponse
	if err := ReadJSON(configFile, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetPrometheusFlags reads Prometheus flags
func (p *Provider) GetPrometheusFlags() (FlagsResponse, error) {
	promPath := p.GetPrometheusCommonPath()
	flagsFile := filepath.Join(promPath, "status", "flags.json")
	var flags FlagsResponse
	if err := ReadJSON(flagsFile, &flags); err != nil {
		return nil, err
	}
	return flags, nil
}

// ReadJSON reads and unmarshals a JSON file
func ReadJSON(filePath string, v any) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filepath.Base(filePath), err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse JSON from %s: %w", filepath.Base(filePath), err)
	}
	return nil
}

// GetReplicaNumbers converts a replica parameter to replica numbers
func GetReplicaNumbers(replicaParam string) []int {
	switch replicaParam {
	case "prometheus-k8s-0", "0":
		return []int{0}
	case "prometheus-k8s-1", "1":
		return []int{1}
	default:
		return []int{0, 1}
	}
}
