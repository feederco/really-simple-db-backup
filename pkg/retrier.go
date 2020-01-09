package pkg

import (
	"log"
	"time"
)

const maxTries = 5

// WithRetry runs runner, and if it returns an error waits again, then tries again
func WithRetry(tag string, runner func() error) error {
	var err error
	for tries := 0; tries < maxTries; tries++ {
		err = runner()
		if err == nil {
			return nil
		}

		waitDuration := getWaitTime(tries)

		log.Printf("%s try: %d, failed with error, sleeping %d\n", tag, tries, waitDuration)

		// Wait exponentially 500 * x^2 milliseconds
		time.Sleep(waitDuration)
	}
	return err
}

func getWaitTime(tries int) time.Duration {
	return time.Duration(500*tries*tries) * time.Millisecond
}
