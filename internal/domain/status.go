package domain

import "time"

const (
	StatusPending = "pending"
	StatusOK      = "ok"
	StatusError   = "error"
)

type SiteStatus struct {
	URL           string     `json:"url"`
	Status        string     `json:"status"` // ok, pending, error
	ResponseCode  *int       `json:"response_code,omitempty"`
	LastCheckTime *time.Time `json:"last_check_time,omitempty"`
	ResponseTime  *int64     `json:"response_time,omitempty"`
	Error         *string    `json:"error,omitempty"`
}
