package mcp

import (
	"fmt"
	"time"

	"github.com/prometheus/common/model"
)

type Response struct {
	Incidents Incidents `json:"incidents"`
}

type Incidents struct {
	Total     int        `json:"total"`
	Incidents []Incident `json:"items"`
}

type Incident struct {
	GroupId   string `json:"id"`
	Severity  string `json:"severity"`
	StartTime string `json:"start_time"`
	Status    string `json:"status"`
	EndTime   string `json:"end_time"`
	Cluster   string `json:"cluster,omitempty"`
	ClusterID string `json:"cluster_id,omitempty"`

	URL                string              `json:"url_details"`
	Alerts             []model.LabelSet    `json:"alerts"`
	AlertsSet          map[string]struct{} `json:"-"`
	AffectedComponents []string            `json:"affected_components"`
	ComponentsSet      map[string]struct{} `json:"-"`
}

// UpdateEndTime updates the end time of the incident following
// the following rules:
// if the new time is zero then set empty string
// if the existing time is empty (incident is active) then do nothing
// if the new time is after existing end time then update it with the
// new time, otherwise do nothing
func (i *Incident) UpdateEndTime(endTime time.Time) error {
	if endTime.IsZero() {
		i.EndTime = ""
		return nil
	}

	if i.EndTime == "" {
		return nil
	}

	existingEndTime, err := time.Parse(time.RFC3339, i.EndTime)
	if err != nil {
		return fmt.Errorf("failed to parse existing end time: %w", err)
	}

	if endTime.After(existingEndTime) {
		i.EndTime = formatToRFC3339(endTime)
	}
	return nil
}

func (i *Incident) UpdateStartTime(startTime time.Time) error {
	existingStartTime, err := time.Parse(time.RFC3339, i.StartTime)
	if err != nil {
		return fmt.Errorf("failed to parse existing start time: %w", err)
	}

	if startTime.Before(existingStartTime) {
		i.StartTime = formatToRFC3339(startTime)
	}
	return nil
}

func (i *Incident) UpdateStatus() {
	if i.EndTime == "" {
		i.Status = "firing"
	} else {
		i.Status = "resolved"
	}
}
