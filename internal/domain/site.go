package domain

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrSiteNotFound = errors.New("site not found")
)

type Site struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CreateSiteRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (req *CreateSiteRequest) Validate() error {
	req.URL = strings.TrimSpace(req.URL)
	req.Name = strings.TrimSpace(req.Name)

	if req.URL == "" {
		return errors.New("url is required but not provided")
	}

	_, err := url.ParseRequestURI(req.URL)
	if err != nil {
		return errors.New("invalid url format")
	}

	return nil
}
