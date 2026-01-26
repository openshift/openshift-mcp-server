package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// buildAlertsParams constructs query parameters for Alertmanager alerts API.
func buildAlertsParams(active, silenced, inhibited bool, filter string) url.Values {
	params := url.Values{}
	params.Set("active", fmt.Sprintf("%t", active))
	params.Set("silenced", fmt.Sprintf("%t", silenced))
	params.Set("inhibited", fmt.Sprintf("%t", inhibited))
	if filter != "" {
		params.Add("filter", filter)
	}
	return params
}

// GetAlerts retrieves alerts from Alertmanager.
func (c *Client) GetAlerts(ctx context.Context, active, silenced, inhibited bool, filter string) ([]Alert, error) {
	body, err := c.executeRequest(ctx, "/api/v2/alerts", buildAlertsParams(active, silenced, inhibited, filter))
	if err != nil {
		return nil, err
	}

	var alerts []Alert
	if err := json.Unmarshal(body, &alerts); err != nil {
		return nil, fmt.Errorf("failed to parse alerts response: %w", err)
	}

	return alerts, nil
}

// GetAlertsRaw retrieves raw JSON alerts from Alertmanager.
func (c *Client) GetAlertsRaw(ctx context.Context, active, silenced, inhibited bool, filter string) ([]byte, error) {
	return c.executeRequest(ctx, "/api/v2/alerts", buildAlertsParams(active, silenced, inhibited, filter))
}
