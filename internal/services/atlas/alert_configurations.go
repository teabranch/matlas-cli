package atlas

import (
	"context"
	"fmt"

	atlasclient "github.com/teabranch/matlas-cli/internal/clients/atlas"
	"github.com/teabranch/matlas-cli/internal/logging"
	"github.com/teabranch/matlas-cli/internal/types"
	admin "go.mongodb.org/atlas-sdk/v20250312006/admin"
)

// AlertConfigurationsService wraps Atlas Alert Configurations API operations.
type AlertConfigurationsService struct {
	client *atlasclient.Client
	logger *logging.Logger
}

// NewAlertConfigurationsService creates a new AlertConfigurationsService.
func NewAlertConfigurationsService(client *atlasclient.Client) *AlertConfigurationsService {
	return &AlertConfigurationsService{
		client: client,
		logger: logging.Default(),
	}
}

// NewAlertConfigurationsServiceWithLogger creates a new AlertConfigurationsService with a custom logger.
func NewAlertConfigurationsServiceWithLogger(client *atlasclient.Client, logger *logging.Logger) *AlertConfigurationsService {
	if logger == nil {
		logger = logging.Default()
	}
	return &AlertConfigurationsService{
		client: client,
		logger: logger,
	}
}

// ListAlertConfigurations returns all alert configurations in the specified project.
func (s *AlertConfigurationsService) ListAlertConfigurations(ctx context.Context, projectID string) ([]admin.GroupAlertsConfig, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}

	s.logger.Debug("Listing alert configurations", "project_id", projectID)

	var configs []admin.GroupAlertsConfig
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertConfigurationsApi.ListAlertConfigurations(ctx, projectID).Execute()
		if err != nil {
			return err
		}
		if resp.Results != nil {
			configs = *resp.Results
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list alert configurations: %w", err)
	}

	s.logger.Debug("Listed alert configurations", "project_id", projectID, "count", len(configs))
	return configs, nil
}

// GetAlertConfiguration returns a specific alert configuration by ID.
func (s *AlertConfigurationsService) GetAlertConfiguration(ctx context.Context, projectID, alertConfigID string) (*admin.GroupAlertsConfig, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	if alertConfigID == "" {
		return nil, fmt.Errorf("alertConfigID required")
	}

	s.logger.Debug("Getting alert configuration", "project_id", projectID, "alert_config_id", alertConfigID)

	var config *admin.GroupAlertsConfig
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertConfigurationsApi.GetAlertConfiguration(ctx, projectID, alertConfigID).Execute()
		if err != nil {
			return err
		}
		config = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get alert configuration %s: %w", alertConfigID, err)
	}

	s.logger.Debug("Got alert configuration", "project_id", projectID, "alert_config_id", alertConfigID)
	return config, nil
}

// CreateAlertConfiguration creates a new alert configuration.
func (s *AlertConfigurationsService) CreateAlertConfiguration(ctx context.Context, projectID string, config *types.AlertConfig) (*admin.GroupAlertsConfig, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	if config == nil {
		return nil, fmt.Errorf("config required")
	}

	s.logger.Debug("Creating alert configuration", "project_id", projectID, "name", config.Metadata.Name)

	atlasConfig, err := s.convertToAtlasConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}

	var result *admin.GroupAlertsConfig
	err = s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertConfigurationsApi.CreateAlertConfiguration(ctx, projectID, atlasConfig).Execute()
		if err != nil {
			return err
		}
		result = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create alert configuration: %w", err)
	}

	s.logger.Debug("Created alert configuration", "project_id", projectID, "name", config.Metadata.Name, "id", result.GetId())
	return result, nil
}

// UpdateAlertConfiguration updates an existing alert configuration.
func (s *AlertConfigurationsService) UpdateAlertConfiguration(ctx context.Context, projectID, alertConfigID string, config *types.AlertConfig) (*admin.GroupAlertsConfig, error) {
	if projectID == "" {
		return nil, fmt.Errorf("projectID required")
	}
	if alertConfigID == "" {
		return nil, fmt.Errorf("alertConfigID required")
	}
	if config == nil {
		return nil, fmt.Errorf("config required")
	}

	s.logger.Debug("Updating alert configuration", "project_id", projectID, "alert_config_id", alertConfigID, "name", config.Metadata.Name)

	atlasConfig, err := s.convertToAtlasConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}

	var result *admin.GroupAlertsConfig
	err = s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertConfigurationsApi.UpdateAlertConfiguration(ctx, projectID, alertConfigID, atlasConfig).Execute()
		if err != nil {
			return err
		}
		result = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to update alert configuration %s: %w", alertConfigID, err)
	}

	s.logger.Debug("Updated alert configuration", "project_id", projectID, "alert_config_id", alertConfigID, "name", config.Metadata.Name)
	return result, nil
}

// DeleteAlertConfiguration deletes an alert configuration.
func (s *AlertConfigurationsService) DeleteAlertConfiguration(ctx context.Context, projectID, alertConfigID string) error {
	if projectID == "" {
		return fmt.Errorf("projectID required")
	}
	if alertConfigID == "" {
		return fmt.Errorf("alertConfigID required")
	}

	s.logger.Debug("Deleting alert configuration", "project_id", projectID, "alert_config_id", alertConfigID)

	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		_, err := api.AlertConfigurationsApi.DeleteAlertConfiguration(ctx, projectID, alertConfigID).Execute()
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to delete alert configuration %s: %w", alertConfigID, err)
	}

	s.logger.Debug("Deleted alert configuration", "project_id", projectID, "alert_config_id", alertConfigID)
	return nil
}

// ListMatcherFieldNames returns all available matcher field names.
func (s *AlertConfigurationsService) ListMatcherFieldNames(ctx context.Context) ([]string, error) {
	s.logger.Debug("Listing matcher field names")

	var fieldNames []string
	err := s.client.Do(ctx, func(api *admin.APIClient) error {
		resp, _, err := api.AlertConfigurationsApi.ListAlertConfigurationMatchersFieldNames(ctx).Execute()
		if err != nil {
			return err
		}
		fieldNames = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list matcher field names: %w", err)
	}

	s.logger.Debug("Listed matcher field names", "count", len(fieldNames))
	return fieldNames, nil
}

// convertToAtlasConfig converts our AlertConfig to Atlas GroupAlertsConfig.
func (s *AlertConfigurationsService) convertToAtlasConfig(config *types.AlertConfig) (*admin.GroupAlertsConfig, error) {
	atlasConfig := admin.NewGroupAlertsConfig()

	// Set basic fields
	if config.Enabled != nil {
		atlasConfig.SetEnabled(*config.Enabled)
	}
	atlasConfig.SetEventTypeName(config.EventTypeName)

	if config.SeverityOverride != "" {
		atlasConfig.SetSeverityOverride(config.SeverityOverride)
	}

	// Convert matchers
	if len(config.Matchers) > 0 {
		matchers := make([]admin.StreamsMatcher, len(config.Matchers))
		for i, matcher := range config.Matchers {
			atlasMatchers := admin.NewStreamsMatcher(matcher.FieldName, matcher.Operator, matcher.Value)
			matchers[i] = *atlasMatchers
		}
		atlasConfig.SetMatchers(matchers)
	}

	// Convert notifications
	if len(config.Notifications) > 0 {
		notifications := make([]admin.AlertsNotificationRootForGroup, len(config.Notifications))
		for i, notification := range config.Notifications {
			atlasNotification, err := s.convertNotification(&notification)
			if err != nil {
				return nil, fmt.Errorf("failed to convert notification %d: %w", i, err)
			}
			notifications[i] = *atlasNotification
		}
		atlasConfig.SetNotifications(notifications)
	}

	// Convert metric threshold
	if config.MetricThreshold != nil {
		threshold := admin.NewFlexClusterMetricThreshold(config.MetricThreshold.MetricName)
		threshold.SetOperator(config.MetricThreshold.Operator)
		threshold.SetThreshold(*config.MetricThreshold.Threshold)
		if config.MetricThreshold.Units != "" {
			threshold.SetUnits(config.MetricThreshold.Units)
		}
		if config.MetricThreshold.Mode != "" {
			threshold.SetMode(config.MetricThreshold.Mode)
		}
		atlasConfig.SetMetricThreshold(*threshold)
	}

	// Convert general threshold
	if config.Threshold != nil {
		threshold := admin.NewStreamProcessorMetricThreshold()
		threshold.SetOperator(config.Threshold.Operator)
		threshold.SetThreshold(*config.Threshold.Threshold)
		if config.Threshold.Units != "" {
			threshold.SetUnits(config.Threshold.Units)
		}
		atlasConfig.SetThreshold(*threshold)
	}

	return atlasConfig, nil
}

// convertNotification converts our AlertNotification to Atlas AlertsNotificationRootForGroup.
func (s *AlertConfigurationsService) convertNotification(notification *types.AlertNotification) (*admin.AlertsNotificationRootForGroup, error) {
	atlasNotification := admin.NewAlertsNotificationRootForGroup()

	atlasNotification.SetTypeName(notification.TypeName)

	if notification.DelayMin != nil {
		atlasNotification.SetDelayMin(*notification.DelayMin)
	}

	if notification.IntervalMin != nil {
		atlasNotification.SetIntervalMin(*notification.IntervalMin)
	}

	// Set type-specific fields
	switch notification.TypeName {
	case "EMAIL":
		if notification.EmailAddress != "" {
			atlasNotification.SetEmailAddress(notification.EmailAddress)
		}
	case "SMS":
		if notification.MobileNumber != "" {
			atlasNotification.SetMobileNumber(notification.MobileNumber)
		}
	case "SLACK":
		if notification.ApiToken != "" {
			atlasNotification.SetApiToken(notification.ApiToken)
		}
		if notification.ChannelName != "" {
			atlasNotification.SetChannelName(notification.ChannelName)
		}
	case "PAGER_DUTY":
		if notification.ServiceKey != "" {
			atlasNotification.SetServiceKey(notification.ServiceKey)
		}
		if notification.Region != "" {
			atlasNotification.SetRegion(notification.Region)
		}
	case "OPS_GENIE":
		if notification.OpsGenieApiKey != "" {
			atlasNotification.SetOpsGenieApiKey(notification.OpsGenieApiKey)
		}
		if notification.OpsGenieRegion != "" {
			atlasNotification.SetOpsGenieRegion(notification.OpsGenieRegion)
		}
	case "DATADOG":
		if notification.DatadogApiKey != "" {
			atlasNotification.SetDatadogApiKey(notification.DatadogApiKey)
		}
		if notification.DatadogRegion != "" {
			atlasNotification.SetDatadogRegion(notification.DatadogRegion)
		}
	case "MICROSOFT_TEAMS":
		if notification.MicrosoftTeamsWebhookUrl != "" {
			atlasNotification.SetMicrosoftTeamsWebhookUrl(notification.MicrosoftTeamsWebhookUrl)
		}
	case "HIP_CHAT":
		if notification.NotificationToken != "" {
			atlasNotification.SetNotificationToken(notification.NotificationToken)
		}
		if notification.RoomName != "" {
			atlasNotification.SetRoomName(notification.RoomName)
		}
	case "WEBHOOK":
		if notification.WebhookUrl != "" {
			atlasNotification.SetWebhookUrl(notification.WebhookUrl)
		}
		if notification.WebhookSecret != "" {
			atlasNotification.SetWebhookSecret(notification.WebhookSecret)
		}
	case "USER", "GROUP", "ORG":
		if notification.EmailEnabled != nil {
			atlasNotification.SetEmailEnabled(*notification.EmailEnabled)
		}
		if notification.SmsEnabled != nil {
			atlasNotification.SetSmsEnabled(*notification.SmsEnabled)
		}
		if len(notification.Roles) > 0 {
			atlasNotification.SetRoles(notification.Roles)
		}
		if notification.Username != "" {
			atlasNotification.SetUsername(notification.Username)
		}
	case "TEAM":
		if notification.TeamId != "" {
			atlasNotification.SetTeamId(notification.TeamId)
		}
		if notification.EmailEnabled != nil {
			atlasNotification.SetEmailEnabled(*notification.EmailEnabled)
		}
		if notification.SmsEnabled != nil {
			atlasNotification.SetSmsEnabled(*notification.SmsEnabled)
		}
	}

	return atlasNotification, nil
}

// ConvertFromAtlasConfig converts Atlas GroupAlertsConfig to our AlertConfig.
func (s *AlertConfigurationsService) ConvertFromAtlasConfig(atlasConfig *admin.GroupAlertsConfig, name string) *types.AlertConfig {
	if atlasConfig == nil {
		return nil
	}

	config := &types.AlertConfig{
		Metadata: types.ResourceMetadata{
			Name: name,
			Labels: map[string]string{
				"atlas-id": atlasConfig.GetId(),
			},
		},
		EventTypeName: atlasConfig.GetEventTypeName(),
	}

	if enabled := atlasConfig.GetEnabled(); enabled {
		config.Enabled = &enabled
	}

	if severity := atlasConfig.GetSeverityOverride(); severity != "" {
		config.SeverityOverride = severity
	}

	// Convert matchers
	if matchers := atlasConfig.GetMatchers(); len(matchers) > 0 {
		config.Matchers = make([]types.AlertMatcher, len(matchers))
		for i, matcher := range matchers {
			config.Matchers[i] = types.AlertMatcher{
				FieldName: matcher.GetFieldName(),
				Operator:  matcher.GetOperator(),
				Value:     matcher.GetValue(),
			}
		}
	}

	// Convert notifications
	if notifications := atlasConfig.GetNotifications(); len(notifications) > 0 {
		config.Notifications = make([]types.AlertNotification, len(notifications))
		for i, notification := range notifications {
			config.Notifications[i] = s.convertFromAtlasNotification(&notification)
		}
	}

	// Convert metric threshold
	if threshold, ok := atlasConfig.GetMetricThresholdOk(); ok && threshold != nil {
		config.MetricThreshold = &types.AlertMetricThreshold{
			MetricName: threshold.GetMetricName(),
			Operator:   threshold.GetOperator(),
			Threshold:  &[]float64{threshold.GetThreshold()}[0],
			Units:      threshold.GetUnits(),
			Mode:       threshold.GetMode(),
		}
	}

	// Convert general threshold
	if threshold, ok := atlasConfig.GetThresholdOk(); ok && threshold != nil {
		config.Threshold = &types.AlertThreshold{
			Operator:  threshold.GetOperator(),
			Threshold: &[]float64{threshold.GetThreshold()}[0],
			Units:     threshold.GetUnits(),
		}
	}

	return config
}

// convertFromAtlasNotification converts Atlas AlertsNotificationRootForGroup to our AlertNotification.
func (s *AlertConfigurationsService) convertFromAtlasNotification(atlasNotification *admin.AlertsNotificationRootForGroup) types.AlertNotification {
	notification := types.AlertNotification{
		TypeName: atlasNotification.GetTypeName(),
	}

	if delayMin := atlasNotification.GetDelayMin(); delayMin != 0 {
		notification.DelayMin = &delayMin
	}

	if intervalMin := atlasNotification.GetIntervalMin(); intervalMin != 0 {
		notification.IntervalMin = &intervalMin
	}

	// Get type-specific fields
	if emailAddress := atlasNotification.GetEmailAddress(); emailAddress != "" {
		notification.EmailAddress = emailAddress
	}

	if emailEnabled := atlasNotification.GetEmailEnabled(); emailEnabled {
		notification.EmailEnabled = &emailEnabled
	}

	if smsEnabled := atlasNotification.GetSmsEnabled(); smsEnabled {
		notification.SmsEnabled = &smsEnabled
	}

	if mobileNumber := atlasNotification.GetMobileNumber(); mobileNumber != "" {
		notification.MobileNumber = mobileNumber
	}

	if channelName := atlasNotification.GetChannelName(); channelName != "" {
		notification.ChannelName = channelName
	}

	if apiToken := atlasNotification.GetApiToken(); apiToken != "" {
		notification.ApiToken = apiToken
	}

	if serviceKey := atlasNotification.GetServiceKey(); serviceKey != "" {
		notification.ServiceKey = serviceKey
	}

	if opsGenieApiKey := atlasNotification.GetOpsGenieApiKey(); opsGenieApiKey != "" {
		notification.OpsGenieApiKey = opsGenieApiKey
	}

	if opsGenieRegion := atlasNotification.GetOpsGenieRegion(); opsGenieRegion != "" {
		notification.OpsGenieRegion = opsGenieRegion
	}

	if datadogApiKey := atlasNotification.GetDatadogApiKey(); datadogApiKey != "" {
		notification.DatadogApiKey = datadogApiKey
	}

	if datadogRegion := atlasNotification.GetDatadogRegion(); datadogRegion != "" {
		notification.DatadogRegion = datadogRegion
	}

	if msTeamsUrl := atlasNotification.GetMicrosoftTeamsWebhookUrl(); msTeamsUrl != "" {
		notification.MicrosoftTeamsWebhookUrl = msTeamsUrl
	}

	if notificationToken := atlasNotification.GetNotificationToken(); notificationToken != "" {
		notification.NotificationToken = notificationToken
	}

	if roomName := atlasNotification.GetRoomName(); roomName != "" {
		notification.RoomName = roomName
	}

	if region := atlasNotification.GetRegion(); region != "" {
		notification.Region = region
	}

	if teamId := atlasNotification.GetTeamId(); teamId != "" {
		notification.TeamId = teamId
	}

	if username := atlasNotification.GetUsername(); username != "" {
		notification.Username = username
	}

	if webhookUrl := atlasNotification.GetWebhookUrl(); webhookUrl != "" {
		notification.WebhookUrl = webhookUrl
	}

	if webhookSecret := atlasNotification.GetWebhookSecret(); webhookSecret != "" {
		notification.WebhookSecret = webhookSecret
	}

	if roles := atlasNotification.GetRoles(); len(roles) > 0 {
		notification.Roles = roles
	}

	return notification
}
