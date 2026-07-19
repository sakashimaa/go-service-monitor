package messaging

import "time"

type SiteCheckEvent struct {
	SiteID       string    `json:"site_id"`
	URL          string    `json:"url"`
	StatusCode   int       `json:"status_code"`
	IsAvailable  bool      `json:"is_available"`
	ResponseTime int64     `json:"response_time_ms"`
	CheckedAt    time.Time `json:"checked_at"`
	ErrorMessage *string   `json:"error_message,omitempty"`
}
