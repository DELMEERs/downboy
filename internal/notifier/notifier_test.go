package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConsoleNotifier(t *testing.T) {
	cn := ConsoleNotifier{}
	cn.NotifySuccess("example.com", 200, 50*time.Millisecond)
	cn.NotifyError("example.com", "connection timeout")
}

func TestDiscordNotifier(t *testing.T) {
	received := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json header, got %s", r.Header.Get("Content-Type"))
		}
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["content"] == nil {
			t.Errorf("expected content field in JSON payload")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dn := NewDiscordNotifier(server.URL)
	dn.NotifyError("example.com", "dial tcp: connection refused")

	time.Sleep(100 * time.Millisecond)
	if !received {
		t.Errorf("expected Discord server to receive HTTP request")
	}
}

func TestTelegramNotifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tn := NewTelegramNotifier("dummy_token", "123456")
	tn.NotifySuccess("example.com", 200, 10*time.Millisecond)
	if tn.botToken != "dummy_token" || tn.chatID != "123456" {
		t.Errorf("telegram credentials mismatch")
	}
}

type mockNotifier struct {
	successCount int
	errorCount   int
}

func (m *mockNotifier) NotifySuccess(url string, statusCode int, duration time.Duration) {
	m.successCount++
}

func (m *mockNotifier) NotifyError(url string, err string) {
	m.errorCount++
}

func TestMultiNotifier(t *testing.T) {
	m1 := &mockNotifier{}
	m2 := &mockNotifier{}

	multi := NewMultiNotifier(m1, m2)
	multi.NotifySuccess("example.com", 200, 10*time.Millisecond)
	multi.NotifyError("example.com", "down")

	if m1.successCount != 1 || m2.successCount != 1 {
		t.Errorf("expected NotifySuccess to be called on all inner notifiers")
	}
	if m1.errorCount != 1 || m2.errorCount != 1 {
		t.Errorf("expected NotifyError to be called on all inner notifiers")
	}
}
