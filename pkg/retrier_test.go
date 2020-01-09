package pkg

import (
	"fmt"
	"testing"
	"time"
)

func TestWithoutRetry(t *testing.T) {
	runs := 0

	err := WithRetry("test", func() error {
		if runs == 0 {
			return nil
		}
		runs++
		return fmt.Errorf("Error %d", runs)
	})

	if err != nil {
		t.Error("Should not return error")
	}

	if runs != 0 {
		t.Error("Expected 0 retries", runs)
	}
}

func TestWithRetry(t *testing.T) {
	runs := 0

	err := WithRetry("test", func() error {
		if runs == 2 {
			return nil
		}
		runs++
		return fmt.Errorf("Error %d", runs)
	})

	if err != nil {
		t.Error("Should not return error")
	}

	if runs != 2 {
		t.Error("Expected only 2 runs, got", runs)
	}
}

func TestRetryTimeout(t *testing.T) {
	results := map[int]time.Duration{
		0: time.Millisecond * 0,
		1: time.Millisecond * 500,
		2: time.Millisecond * 2000,
		3: time.Millisecond * 4500,
		4: time.Millisecond * 8000,
		5: time.Millisecond * 12500,
		6: time.Millisecond * 18000,
	}

	for try, expected := range results {
		if timeout := getWaitTime(try); timeout != expected {
			t.Errorf("Try %d: Expected %d got %d", try, expected, timeout)
		}
	}
}
