package transport

import (
	"lan-clipboard-sync/internal/core"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReceiveAuthenticationAndDeduplication(t *testing.T) {
	count := 0
	n, err := New(Config{ID: "self", Name: "Self", Secret: "long-enough-key", Port: 8080, DiscoveryPort: 8081}, func(core.Message) error { count++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(http.HandlerFunc(n.receive))
	defer s.Close()
	m := core.Message{ID: "id", SenderID: "other", Sender: "Other", Text: "hello", Created: time.Now()}
	payload, _ := core.Encode(m)
	req, _ := http.NewRequest(http.MethodPost, s.URL, strings.NewReader(string(payload)))
	req.Header.Set("X-LanSync-Signature", core.Sign("long-enough-key", payload))
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 204 {
		t.Fatalf("first receive: %v %v", err, resp)
	}
	resp.Body.Close()
	req, _ = http.NewRequest(http.MethodPost, s.URL, strings.NewReader(string(payload)))
	req.Header.Set("X-LanSync-Signature", core.Sign("long-enough-key", payload))
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()
	if count != 1 {
		t.Fatalf("handled %d messages", count)
	}
}

func TestReceiveCanRetryAfterHandlerFailure(t *testing.T) {
	attempts := 0
	n, err := New(Config{ID: "self", Name: "Self", Secret: "long-enough-key", Port: 8080, DiscoveryPort: 8081}, func(core.Message) error {
		attempts++
		if attempts == 1 {
			return assertErr{}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(http.HandlerFunc(n.receive))
	defer s.Close()
	m := core.Message{ID: "retry-id", SenderID: "other", Sender: "Other", Text: "hello", Created: time.Now()}
	payload, _ := core.Encode(m)
	for _, expected := range []int{http.StatusInternalServerError, http.StatusNoContent} {
		req, _ := http.NewRequest(http.MethodPost, s.URL, strings.NewReader(string(payload)))
		req.Header.Set("X-LanSync-Signature", core.Sign("long-enough-key", payload))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != expected {
			t.Fatalf("status %d, want %d", resp.StatusCode, expected)
		}
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "expected handler failure" }
