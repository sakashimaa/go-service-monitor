package checker

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sakashimaa/site-monitor/internal/config"
)

func TestCheckSite(t *testing.T) {
	tests := []struct {
		name          string
		serverFunc    http.HandlerFunc
		timeout       time.Duration
		wantAvailable bool
		wantCode      int
		wantErr       bool
	}{
		{
			name: "успешный ответ 200",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			timeout:       time.Second,
			wantAvailable: true,
			wantCode:      http.StatusOK,
			wantErr:       false,
		},
		{
			name: "клиентская ошибка 404",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			timeout:       time.Second,
			wantAvailable: false,
			wantCode:      http.StatusNotFound,
			wantErr:       false,
		},
		{
			name: "серверная ошибка 500",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			timeout:       time.Second,
			wantAvailable: false,
			wantCode:      http.StatusInternalServerError,
			wantErr:       false,
		},
		{
			name: "редирект 302 не считается доступным (только 200 == ok)",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusFound)
			},
			timeout:       time.Second,
			wantAvailable: false,
			wantCode:      http.StatusFound,
			wantErr:       false,
		},
		{
			name: "таймаут: хендлер отвечает дольше клиентского таймаута",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			timeout:       50 * time.Millisecond,
			wantAvailable: false,
			wantCode:      0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverFunc)
			defer server.Close()

			cfg := &config.Config{Timeout: tt.timeout}

			start := time.Now()
			res := CheckSite(server.URL, cfg)
			elapsed := time.Since(start)

			if res.AvailableStatus != tt.wantAvailable {
				t.Errorf("Available status = %v, wanted %v", res.AvailableStatus, tt.wantAvailable)
			}

			if res.ResponseCode != tt.wantCode {
				t.Errorf("Resp code = %v, wanted %v", res.ResponseCode, tt.wantCode)
			}

			if tt.wantErr && res.Error == nil {
				t.Error("wanted error, got nil")
			}

			if !tt.wantErr && res.Error != nil {
				t.Errorf("error was not awaited, got %v", res.Error)
			}

			if res.ResponseTime <= 0 {
				t.Errorf("responseTime must be >= 0, got %v", res.ResponseTime)
			}

			if res.ResponseTime > elapsed+50*time.Millisecond {
				t.Errorf("ResponseTime %v differentiate to much with real elapsed %v", res.ResponseTime, elapsed)
			}

			if res.URL != server.URL {
				t.Errorf("URL = %v, want %v", res.URL, server.URL)
			}
		})
	}
}

func TestCheckSite_ConnectionRefused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	url := server.URL
	server.Close()

	cfg := &config.Config{Timeout: time.Second}

	res := CheckSite(url, cfg)

	if res.AvailableStatus {
		t.Error("AvailableStatus = true, want false")
	}

	if res.Error == nil {
		t.Fatal("wanted error, connection to closed server")
	}

	if res.ResponseCode != 0 {
		t.Errorf("Response code %d, want 0", res.ResponseCode)
	}

	if res.ResponseTime <= 0 {
		t.Errorf("ResponseTime must be > 0 even with closed server, got %v", res.ResponseTime)
	}
}
