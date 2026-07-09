package domain

import (
	"errors"
	"time"
)

var (
	ErrSiteHistoryNotFound = errors.New("no check history for site found")
)

type CheckHistory struct {
	ID           string    `json:"id"`
	SiteID       string    `json:"site_id"`
	Status       string    `json:"status"`
	ResponseCode int       `json:"response_code"`
	ResponseTime int64     `json:"response_time_ms"`
	Error        *string   `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type SiteHistoryResponse struct {
	Data       []CheckHistory `json:"data"`
	NextCursor *time.Time     `json:"next_cursor,omitempty"`
	HasMore    bool           `json:"has_more"`
}
