package checker

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type spyNotifier struct {
	lastURL       string
	lastStatus    int
	lastError     string
	successCalled bool
	errorCalled   bool
}

func (s *spyNotifier) NotifySuccess(url string, statusCode int, duration time.Duration) {
	s.lastURL = url
	s.lastStatus = statusCode
	s.successCalled = true
}

func (s *spyNotifier) NotifyError(url string, err string) {
	s.lastURL = url
	s.lastError = err
	s.errorCalled = true
}

func TestCheckURLInvalidDomain(t *testing.T) {
	invalidURL := "absolutely-fake-website.notexist"
	spy := &spyNotifier{}
	var wg sync.WaitGroup

	wg.Add(1)
	result := CheckURL(invalidURL, &wg, spy)

	if result != false {
		t.Errorf("expected CheckURL to return false for invalid domain got true")
	}
	if !spy.errorCalled {
		t.Errorf("expected NotifyError to be called but it wasnt")
	}
	expectedCleanURL := "absolutely-fake-website.notexist"
	if spy.lastURL != expectedCleanURL {
		t.Errorf("expected clean URL to be %q got %q", expectedCleanURL, spy.lastURL)
	}
	if spy.lastError != "no such host" {
		t.Errorf("expected error message to contain 'no such host' got %q", spy.lastError)
	}
}

func TestCheckURLSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testURL := server.URL
	spy := &spyNotifier{}
	var wg sync.WaitGroup

	wg.Add(1)

	result := CheckURL(testURL, &wg, spy)

	if result != true {
		t.Errorf("expected CheckURL to return true for working server got false")
	}
	if !spy.successCalled {
		t.Errorf("expected NotifySuccess to be called but it wasnt")
	}
	if spy.lastStatus != http.StatusOK {
		t.Errorf("expected notified status to be 200 got %d", spy.lastStatus)
	}

	if spy.lastURL == testURL {
		t.Errorf("expected protocol prefix to be stripped from URL in notification but it stayed: %q", spy.lastURL)
	}
}
