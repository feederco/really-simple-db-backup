package pkg

import (
	"fmt"
	"os"
)

// AlertError alerts an error to the system administrator
func AlertError(message string, err error) {
	hostname, _ := os.Hostname()
	fmt.Printf("!U=?!?!?!?!?!? [host: %s] %s with error %s\n", hostname, message, err)
}
