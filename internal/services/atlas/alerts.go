package atlas

import (
	"context"
	"fmt"
	"time"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/types"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

// AlertsService wraps Atlas Alerts API operations.
type AlertsService struct {
	client *atlasclient.Client
	logger *logging.Logger
}

// NewAlertsService creates a new AlertsService.
func NewAlertsService(client *atlasclient.Client) *AlertsService {
	return &AlertsService{
		client: client,
		logger: logging.Default(),
	}
}

// NewAlertsServiceWithLogger creates a new AlertsService with a custom logger.
func NewAlertsServiceWithLogger(client *atlasclient.Client, logger *logging.Logger) *AlertsService {
	if logger == nil {
		logger = logging.Default()
	}
	return &AlertsService{
		client: client,
		logger: logger,
	}
}

// ListAlerts returns all alerts in the specified project.
func (s *AlertsService) ListAlerts(ctx context.Context, projectID string) ([]admin.AlertViewForNdsGroup, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	s.logger.Debug("Listing alerts", "project_id", projectID)

	var alerts []admin.AlertViewForNdsGroup
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertsApi.ListAlerts(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		if resp.Results != nil {
			alerts = *resp.Results
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}

	s.logger.Debug("Listed alerts", "project_id", projectID, "count", len(alerts))
	return alerts, nil
}

// GetAlert returns a specific alert by ID.
func (s *AlertsService) GetAlert(ctx context.Context, projectID, alertID string) (*admin.AlertViewForNdsGroup, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	if alertID == "" {
		return nil, fmt.Errorf("alertID required")
	}

	s.logger.Debug("Getting alert", "project_id", projectID, "alert_id", alertID)

	var alert *admin.AlertViewForNdsGroup
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertsApi.GetAlert(ctx, projectID, alertID).Execute()
		if err != nil {
			return err
		}
		alert = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get alert %s: %w", alertID, err)
	}

	s.logger.Debug("Got alert", "project_id", projectID, "alert_id", alertID)
	return alert, nil
}

// AcknowledgeAlert acknowledges or unacknowledges an alert.
func (s *AlertsService) AcknowledgeAlert(ctx context.Context, projectID, alertID string, acknowledge bool, until *string) (*admin.AlertViewForNdsGroup, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	if alertID == "" {
		return nil, fmt.Errorf("alertID required")
	}

	s.logger.Debug("Acknowledging alert", "project_id", projectID, "alert_id", alertID, "acknowledge", acknowledge)

	acknowledgeAlert := admin.NewAcknowledgeAlert()
	if acknowledge && until != nil {
		// Parse the until time string and set it
		if untilTime, err := time.Parse(time.RFC3339, *until); err == nil {
			acknowledgeAlert.SetAcknowledgedUntil(untilTime)
		}
	}

	var alert *admin.AlertViewForNdsGroup
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertsApi.AcknowledgeAlert(ctx, projectID, alertID, acknowledgeAlert).Execute()
		if err != nil {
			return err
		}
		alert = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to acknowledge alert %s: %w", alertID, err)
	}

	s.logger.Debug("Acknowledged alert", "project_id", projectID, "alert_id", alertID, "acknowledge", acknowledge)
	return alert, nil
}

// ListAlertsByConfiguration returns all alerts for a specific alert configuration.
func (s *AlertsService) ListAlertsByConfiguration(ctx context.Context, projectID, alertConfigID string) ([]admin.AlertViewForNdsGroup, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	if alertConfigID == "" {
		return nil, fmt.Errorf("alertConfigID required")
	}

	s.logger.Debug("Listing alerts by configuration", "project_id", projectID, "alert_config_id", alertConfigID)

	var alerts []admin.AlertViewForNdsGroup
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertsApi.ListAlertsByAlertConfigurationId(ctx, projectID, alertConfigID).Execute()
		if err != nil {
			return err
		}
		if resp.Results != nil {
			alerts = *resp.Results
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list alerts by configuration %s: %w", alertConfigID, err)
	}

	s.logger.Debug("Listed alerts by configuration", "project_id", projectID, "alert_config_id", alertConfigID, "count", len(alerts))
	return alerts, nil
}

// ConvertAlertToStatus converts an Atlas alert to our AlertStatus type.
func (s *AlertsService) ConvertAlertToStatus(alert *admin.AlertViewForNdsGroup) *types.AlertStatus {
	if alert == nil {
		return nil
	}

	status := &types.AlertStatus{
		EventTypeName: alert.GetEventTypeName(),
		Status:        alert.GetStatus(),
	}

	if id := alert.GetId(); id != "" {
		status.ID = id
	}

	if configID := alert.GetAlertConfigId(); configID != "" {
		status.AlertConfigID = configID
	}

	if ackUntil := alert.GetAcknowledgedUntil(); !ackUntil.IsZero() {
		status.AcknowledgedUntil = &ackUntil
	}

	if ackUser := alert.GetAcknowledgingUsername(); ackUser != "" {
		status.AcknowledgingUser = ackUser
	}

	if created := alert.GetCreated(); !created.IsZero() {
		status.Created = &created
	}

	if updated := alert.GetUpdated(); !updated.IsZero() {
		status.Updated = &updated
	}

	if lastNotified := alert.GetLastNotified(); !lastNotified.IsZero() {
		status.LastNotified = &lastNotified
	}

	if metricName := alert.GetMetricName(); metricName != "" {
		status.MetricName = metricName
	}

	if hostname := alert.GetHostnameAndPort(); hostname != "" {
		status.HostnameAndPort = hostname
	}

	if replicaSet := alert.GetReplicaSetName(); replicaSet != "" {
		status.ReplicaSetName = replicaSet
	}

	if clusterName := alert.GetClusterName(); clusterName != "" {
		status.ClusterName = clusterName
	}

	if currentValue, ok := alert.GetCurrentValueOk(); ok && currentValue != nil {
		status.CurrentValue = &types.AlertCurrentValue{}
		if number := currentValue.GetNumber(); number != 0 {
			status.CurrentValue.Number = &number
		}
		if units := currentValue.GetUnits(); units != "" {
			status.CurrentValue.Units = units
		}
	}

	return status
}
