package mustgather

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetETCDHealth reads ETCD health from the archive
func (p *Provider) GetETCDHealth() (*ETCDHealth, error) {
	etcdDir := filepath.Join(p.metadata.ContainerDir, "etcd_info")

	health := &ETCDHealth{Healthy: true}

	// Read endpoint health
	healthFile := filepath.Join(etcdDir, "endpoint_health.json")
	healthData, err := os.ReadFile(healthFile)
	if err != nil {
		return nil, fmt.Errorf("ETCD health data not found: %w", err)
	}

	var endpoints []struct {
		Endpoint string `json:"endpoint"`
		Health   string `json:"health"`
	}
	if err := json.Unmarshal(healthData, &endpoints); err != nil {
		return nil, fmt.Errorf("failed to parse ETCD health: %w", err)
	}

	for _, ep := range endpoints {
		health.Endpoints = append(health.Endpoints, ETCDEndpoint{
			Address: ep.Endpoint,
			Health:  ep.Health,
		})
		if ep.Health != "true" && ep.Health != "healthy" {
			health.Healthy = false
		}
	}

	// Read alarms
	alarmFile := filepath.Join(etcdDir, "alarm_list.json")
	if alarmData, err := os.ReadFile(alarmFile); err == nil {
		var alarmResponse struct {
			Alarms []struct {
				MemberID uint64 `json:"memberID"`
				Alarm    string `json:"alarm"`
			} `json:"alarms"`
		}
		if err := json.Unmarshal(alarmData, &alarmResponse); err == nil {
			for _, alarm := range alarmResponse.Alarms {
				health.Alarms = append(health.Alarms, fmt.Sprintf("Member %d: %s", alarm.MemberID, alarm.Alarm))
				health.Healthy = false
			}
		}
	}

	return health, nil
}

// GetETCDObjectCount reads ETCD object counts from the archive
func (p *Provider) GetETCDObjectCount() (map[string]int64, error) {
	countFile := filepath.Join(p.metadata.ContainerDir, "etcd_info", "object_count.json")
	data, err := os.ReadFile(countFile)
	if err != nil {
		return nil, fmt.Errorf("ETCD object count data not found: %w", err)
	}

	var counts map[string]int64
	if err := json.Unmarshal(data, &counts); err != nil {
		return nil, fmt.Errorf("failed to parse ETCD object count: %w", err)
	}

	return counts, nil
}

// ReadETCDFile reads a raw ETCD info JSON file and returns its content
func (p *Provider) ReadETCDFile(filename string) ([]byte, error) {
	filePath := filepath.Join(p.metadata.ContainerDir, "etcd_info", filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("ETCD file %s not found: %w", filename, err)
	}
	return data, nil
}
