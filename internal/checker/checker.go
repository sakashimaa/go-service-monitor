package checker

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type Result struct {
	URL             string
	ResponseCode    int
	AvailableStatus bool
	Error           error
}

func CheckSite(url string) Result {
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return Result{
			URL:             url,
			ResponseCode:    0,
			AvailableStatus: false,
			Error:           fmt.Errorf("failed to make request: %w", err),
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v\n", err)
		}
	}()

	return Result{
		URL:             url,
		ResponseCode:    resp.StatusCode,
		AvailableStatus: resp.StatusCode == http.StatusOK,
		Error:           nil,
	}
}
