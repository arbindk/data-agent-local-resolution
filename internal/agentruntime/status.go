package agentruntime

import (
	"sync"
	"time"
)

type RuntimeStatus struct {
	ServiceName       string    `json:"service_name"`
	PollingEnabled    bool      `json:"polling_enabled"`
	PollIntervalSec   int       `json:"poll_interval_seconds"`
	LastPollAt        time.Time `json:"last_poll_at,omitempty"`
	LastPollHTTPCode  int       `json:"last_poll_http_code,omitempty"`
	LastPollStatus    string    `json:"last_poll_status,omitempty"`
	LastPollMessage   string    `json:"last_poll_message,omitempty"`
	LastPollTargetURL string    `json:"last_poll_target_url,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

var (
	mu     sync.Mutex
	status = RuntimeStatus{
		ServiceName: "Everest Data Agent",
		UpdatedAt:   time.Now().UTC(),
	}
)

func ConfigurePolling(enabled bool, intervalSec int, targetURL string) {
	mu.Lock()
	defer mu.Unlock()

	status.PollingEnabled = enabled
	status.PollIntervalSec = intervalSec
	status.LastPollTargetURL = targetURL
	status.UpdatedAt = time.Now().UTC()
}

func RecordPollResult(httpCode int, pollStatus string, message string) {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().UTC()
	status.LastPollAt = now
	status.LastPollHTTPCode = httpCode
	status.LastPollStatus = pollStatus
	status.LastPollMessage = message
	status.UpdatedAt = now
}

func CurrentStatus() RuntimeStatus {
	mu.Lock()
	defer mu.Unlock()

	return status
}
