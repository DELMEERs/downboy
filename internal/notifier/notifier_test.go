package notifier

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConsoleNotifier(_ *testing.T) {
	cn := ConsoleNotifier{}
	cn.NotifySuccess("example.com", 200, 50*time.Millisecond)
	cn.NotifyError("example.com", "connection timeout")
}

func TestConsoleNotifier_SpinnerFrames(t *testing.T) {
	expected := []string{"[   ]", "[-  ]", "[-- ]", "[---]", "[ --]", "[  -]"}
	if len(SpinnerFrames) != len(expected) {
		t.Fatalf("expected %d spinner frames, got %d", len(expected), len(SpinnerFrames))
	}
	for i, frame := range SpinnerFrames {
		if frame != expected[i] {
			t.Errorf("frame %d expected %q, got %q", i, expected[i], frame)
		}
	}
}

func TestConsoleNotifier_ImmediateRender(t *testing.T) {
	var buf bytes.Buffer
	urls := []string{"google.com", "github.com"}
	cn := NewConsoleNotifierWithWriter(urls, &buf)
	defer cn.Stop()

	out := buf.String()
	if !strings.Contains(out, "google.com") || !strings.Contains(out, "github.com") {
		t.Errorf("expected immediate rendering of target URLs, got: %q", out)
	}
	if !strings.Contains(out, "[   ]") {
		t.Errorf("expected initial bracket [   ], got: %q", out)
	}
}

func TestConsoleNotifier_CompletionAndSorting(t *testing.T) {
	var buf bytes.Buffer
	urls := []string{"google.com", "badurl.invalid", "github.com"}
	cn := NewConsoleNotifierWithWriter(urls, &buf)

	cn.NotifySuccess("google.com", 200, 15*time.Millisecond)
	cn.NotifyError("badurl.invalid", "no such host")
	cn.NotifySuccess("github.com", 200, 20*time.Millisecond)

	cn.Stop()

	out := buf.String()
	if !strings.Contains(out, "[ ✓ ]") {
		t.Errorf("expected success bracket [ ✓ ] in output, got: %q", out)
	}
	if !strings.Contains(out, "[ X ]") {
		t.Errorf("expected failure bracket [ X ] in output, got: %q", out)
	}

	lines := strings.Split(out, "\n")
	var finalLines []string
	for _, l := range lines {
		if strings.Contains(l, "[ X ]") || strings.Contains(l, "[ ✓ ]") {
			finalLines = append(finalLines, l)
		}
	}

	if len(finalLines) < 3 {
		t.Fatalf("expected at least 3 status lines in output, got %d", len(finalLines))
	}

	lastThree := finalLines[len(finalLines)-3:]
	if !strings.Contains(lastThree[0], "badurl.invalid") || !strings.Contains(lastThree[0], "[ X ]") {
		t.Errorf("expected failed check (badurl.invalid) at top of final output, got: %q", lastThree[0])
	}
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

func (m *mockNotifier) NotifySuccess(_ string, _ int, _ time.Duration) {
	m.successCount++
}

func (m *mockNotifier) NotifyError(_ string, _ string) {
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
