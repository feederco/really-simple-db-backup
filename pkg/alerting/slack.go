package alerting

import (
	"bytes"
	"encoding/json"
	"net/http"
)

const slackDefaultUsername = "BackupsBot"
const slackDefaultIconEmoji = ":card_file_box:"

// SlackConfig contains config values for slack config
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
	IconEmoji  string `json:"icon_emoji"`
}

type slackWebhook struct {
	Username  string `json:"username"`
	Text      string `json:"text"`
	IconEmoji string `json:"icon_emoji"`
}

// SlackLog log a message to my slack
func SlackLog(message string, config *SlackConfig) error {
	username := config.Username
	if username == "" {
		username = slackDefaultUsername
	}

	iconEmoji := config.IconEmoji
	if iconEmoji == "" {
		iconEmoji = slackDefaultIconEmoji
	}

	data := slackWebhook{
		Username:  username,
		Text:      message,
		IconEmoji: iconEmoji,
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	body := bytes.NewReader(payloadBytes)
	resp, err := http.Post(config.WebhookURL, "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
