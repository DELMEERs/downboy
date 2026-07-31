package checker

import (
	"errors"
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

func withMockHead(t *testing.T, fn func(url string) (*http.Response, error)) {
	t.Helper()
	original := httpHead
	httpHead = fn
	t.Cleanup(func() {
		httpHead = original
	})
}

func TestCheckURL_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		inputURL      string
		mockErr       error
		expectedClean string
		expectedError string
	}{
		{
			name:          "no such host, no scheme given",
			inputURL:      "absolutely-fake-website.notexist",
			mockErr:       errors.New(`Head "http://absolutely-fake-website.notexist": dial tcp: lookup absolutely-fake-website.notexist: no such host`),
			expectedClean: "absolutely-fake-website.notexist",
			expectedError: "no such host",
		},
		{
			name:          "connection refused, https scheme given",
			inputURL:      "https://unreachable.example",
			mockErr:       errors.New(`Head "https://unreachable.example": dial tcp: connect: connection refused`),
			expectedClean: "unreachable.example",
			expectedError: "connection refused",
		},
		{
			name:          "timeout, http scheme given explicitly",
			inputURL:      "http://slow.example",
			mockErr:       errors.New(`Head "http://slow.example": context deadline exceeded: timeout`),
			expectedClean: "slow.example",
			expectedError: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withMockHead(t, func(url string) (*http.Response, error) {
				return nil, tt.mockErr
			})

			spy := &spyNotifier{}
			var wg sync.WaitGroup
			wg.Add(1)

			result := CheckURL(tt.inputURL, &wg, spy)

			if result != false {
				t.Errorf("expected CheckURL to return false got true")
			}
			if !spy.errorCalled {
				t.Fatal("expected NotifyError to be called but it wasnt")
			}
			if spy.successCalled {
				t.Errorf("expected NotifySuccess NOT to be called but it was")
			}
			if spy.lastURL != tt.expectedClean {
				t.Errorf("expected clean URL %q got %q", tt.expectedClean, spy.lastURL)
			}
			if spy.lastError != tt.expectedError {
				t.Errorf("expected error message %q got %q", tt.expectedError, spy.lastError)
			}
		})
	}
}

func TestCheckURL_SuccessCases(t *testing.T) {
	tests := []struct {
		name          string
		inputURL      string
		statusCode    int
		expectedClean string
	}{
		{
			name:          "no scheme given, defaults to http",
			inputURL:      "example.com",
			statusCode:    http.StatusOK,
			expectedClean: "example.com",
		},
		{
			name:          "https scheme given explicitly",
			inputURL:      "https://example.com",
			statusCode:    http.StatusOK,
			expectedClean: "example.com",
		},
		{
			name:          "non-200 status is still reported as success",
			inputURL:      "http://example.com",
			statusCode:    http.StatusNotFound,
			expectedClean: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			withMockHead(t, func(url string) (*http.Response, error) {
				capturedURL = url
				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       http.NoBody,
				}, nil
			})

			spy := &spyNotifier{}
			var wg sync.WaitGroup
			wg.Add(1)

			result := CheckURL(tt.inputURL, &wg, spy)

			if result != true {
				t.Errorf("expected CheckURL to return true got false")
			}
			if !spy.successCalled {
				t.Fatal("expected NotifySuccess to be called but it wasnt")
			}
			if spy.errorCalled {
				t.Errorf("expected NotifyError NOT to be called but it was")
			}
			if spy.lastStatus != tt.statusCode {
				t.Errorf("expected status %d got %d", tt.statusCode, spy.lastStatus)
			}
			if spy.lastURL != tt.expectedClean {
				t.Errorf("expected clean URL %q got %q", tt.expectedClean, spy.lastURL)
			}

			if capturedURL != "http://"+tt.expectedClean && capturedURL != "https://"+tt.expectedClean {
				t.Errorf("expected request URL to have a scheme got %q", capturedURL)
			}
		})
	}
}

func TestCheckURL_NilWaitGroup(t *testing.T) {
	withMockHead(t, func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	spy := &spyNotifier{}

	result := CheckURL("http://example.com", nil, spy)

	if result != true {
		t.Errorf("expected CheckURL to return true got false")
	}
	if !spy.successCalled {
		t.Errorf("expected NotifySuccess to be called but it wasnt")
	}
}

func TestCheckURL_RealServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	spy := &spyNotifier{}
	var wg sync.WaitGroup
	wg.Add(1)

	result := CheckURL(server.URL, &wg, spy)

	if result != true {
		t.Errorf("expected CheckURL to return true for working server got false")
	}
	if !spy.successCalled {
		t.Errorf("expected NotifySuccess to be called but it wasnt")
	}
	if spy.lastStatus != http.StatusOK {
		t.Errorf("expected notified status to be 200 got %d", spy.lastStatus)
	}
	if spy.lastURL == server.URL {
		t.Errorf("expected protocol prefix to be stripped from URL in notification but it stayed: %q", spy.lastURL)
	}
}

func TestCheckURLWithOptions_RealServer405Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer server.Close()

	spy := &spyNotifier{}
	opts := CheckOptions{
		Timeout: 2 * time.Second,
		Retries: 0,
	}

	res := CheckURLWithOptions(t.Context(), server.URL, nil, spy, opts)
	if !res.IsUp {
		t.Errorf("expected IsUp true on 405 fallback to GET, got false")
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status code 200 after GET fallback, got %d", res.StatusCode)
	}
}

func TestCheckURLWithOptions_Retries(t *testing.T) {
	attempts := 0
	withMockHead(t, func(url string) (*http.Response, error) {
		attempts++
		if attempts < 2 {
			return nil, errors.New("temporary failure")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	spy := &spyNotifier{}
	opts := CheckOptions{
		Timeout: 1 * time.Second,
		Retries: 2,
	}

	res := CheckURLWithOptions(t.Context(), "http://retry.example", nil, spy, opts)
	if !res.IsUp {
		t.Errorf("expected IsUp true after retry, got false")
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}
