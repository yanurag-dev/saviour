package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GoogleChatNotifier sends alerts to Google Chat via webhook
type GoogleChatNotifier struct {
	webhookURL   string
	dashboardURL string
	httpClient   *http.Client
}

// NewGoogleChatNotifier creates a new Google Chat notifier
func NewGoogleChatNotifier(webhookURL, dashboardURL string) *GoogleChatNotifier {
	return &GoogleChatNotifier{
		webhookURL:   webhookURL,
		dashboardURL: dashboardURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendAlert sends an alert to Google Chat
func (g *GoogleChatNotifier) SendAlert(alert *Alert) error {
	message := g.buildMessage(alert)

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Google Chat message: %w", err)
	}

	resp, err := g.httpClient.Post(g.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send Google Chat webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Google Chat webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// buildMessage creates a Google Chat card message
func (g *GoogleChatNotifier) buildMessage(alert *Alert) map[string]interface{} {
	// Determine icon based on severity
	icon := g.getSeverityIcon(alert.Severity)

	// Build sections
	sections := []map[string]interface{}{
		{
			"widgets": []map[string]interface{}{
				{
					"textParagraph": map[string]interface{}{
						"text": fmt.Sprintf("<b>%s</b>", alert.Message),
					},
				},
				{
					"keyValue": map[string]interface{}{
						"topLabel": "Alert Type",
						"content":  alert.AlertType,
					},
				},
				{
					"keyValue": map[string]interface{}{
						"topLabel": "Severity",
						"content":  alert.Severity,
					},
				},
				{
					"keyValue": map[string]interface{}{
						"topLabel": "Triggered At",
						"content":  alert.TriggeredAt.Format("2006-01-02 15:04:05 MST"),
					},
				},
			},
		},
	}

	// Add dashboard link if available
	if g.dashboardURL != "" {
		buttons := map[string]interface{}{
			"buttons": []map[string]interface{}{
				{
					"textButton": map[string]interface{}{
						"text": "View Dashboard",
						"onClick": map[string]interface{}{
							"openLink": map[string]interface{}{
								"url": g.dashboardURL,
							},
						},
					},
				},
			},
		}
		sections = append(sections, buttons)
	}

	// Build card
	card := map[string]interface{}{
		"cards": []map[string]interface{}{
			{
				"header": map[string]interface{}{
					"title":    fmt.Sprintf("%s %s Alert", icon, alert.Severity),
					"subtitle": alert.AgentName,
				},
				"sections": sections,
			},
		},
	}

	// Add thread key for grouping related alerts
	if g.supportsThreading() {
		card["thread"] = map[string]interface{}{
			"threadKey": fmt.Sprintf("alert-%s-%s", alert.AgentName, alert.AlertType),
		}
	}

	return card
}

// getSeverityIcon returns emoji icon based on severity
func (g *GoogleChatNotifier) getSeverityIcon(severity string) string {
	switch severity {
	case "critical":
		return "🚨"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	default:
		return "📢"
	}
}

// supportsThreading checks if webhook supports threading
func (g *GoogleChatNotifier) supportsThreading() bool {
	// Google Chat webhooks support threading
	return true
}

// ConsoleNotifier logs alerts to console (for testing)
type ConsoleNotifier struct{}

// NewConsoleNotifier creates a console notifier
func NewConsoleNotifier() *ConsoleNotifier {
	return &ConsoleNotifier{}
}

// SendAlert logs the alert to console
func (c *ConsoleNotifier) SendAlert(alert *Alert) error {
	fmt.Printf("\n=== ALERT ===\n")
	fmt.Printf("Type: %s\n", alert.AlertType)
	fmt.Printf("Severity: %s\n", alert.Severity)
	fmt.Printf("Agent: %s\n", alert.AgentName)
	fmt.Printf("Message: %s\n", alert.Message)
	fmt.Printf("Triggered: %s\n", alert.TriggeredAt.Format(time.RFC3339))
	fmt.Printf("=============\n\n")
	return nil
}
