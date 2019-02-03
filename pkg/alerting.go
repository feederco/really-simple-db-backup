package pkg

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/feederco/really-simple-db-backup/pkg/alerting"
)

// AlertingConfig sub-config type for alerting related
type AlertingConfig struct {
	Slack *alerting.SlackConfig `json:"slack"`
}

// AlertError alerts an error to the system administrator
func AlertError(alertingConfig *AlertingConfig, message string, err error) {
	hostname, _ := os.Hostname()

	buf := make([]byte, 3000)
	buf = buf[:runtime.Stack(buf, false)]

	fullMessage := fmt.Sprintf("[*BACKUP FAILURE*] [%s] [host: `%s`] `%s` with error: `%s`\n", time.Now().Format(time.RFC3339), hostname, message, err)
	fullMessageWithStack := fullMessage + "\n\n```\n" + string(buf) + "```"

	if alertingConfig != nil && alertingConfig.Slack != nil {
		err := alerting.SlackLog(fullMessageWithStack, alertingConfig.Slack)

		if err != nil {
			ErrorLog.Println("Warning: Could not alert to Slack.", err)
			ErrorLog.Println("Original error", fullMessage)
		}
	}

	// Always print to error log
	ErrorLog.Print(fullMessage)
}

// AlertMessage simply alerts a message to the correct channels
func AlertMessage(alertingConfig *AlertingConfig, message string) {
	hostname, _ := os.Hostname()

	fullMessage := fmt.Sprintf("[backup message] [%s] [host: %s] %s", time.Now().Format(time.RFC3339), hostname, message)

	if alertingConfig != nil && alertingConfig.Slack != nil {
		err := alerting.SlackLog(fullMessage, alertingConfig.Slack)

		if err != nil {
			Log.Println("Warning: Could not alert to Slack.", err)
		}
	}

	// Always print to error log
	Log.Print(fullMessage)
}
