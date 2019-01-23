package pkg

import (
	"os"
)

// AlertError alerts an error to the system administrator
func AlertError(message string, err error) {
	hostname, _ := os.Hostname()
	ErrorLog.Printf("[ALERT] [host: %s] %s with error %s\n", hostname, message, err)
}
