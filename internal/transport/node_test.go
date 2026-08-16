package transport

import (
	"encoding/json"
	"lan-clipboard-sync/internal/core"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestRecordAnnouncementUsesAnnouncedPort(t *testing.T) {
	n, err := New(Config{ID: "self", Name: "Self", Secret: "long-enough-key", Port: 8080, DiscoveryPort: 8081}, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, port := range []int{1, 43210, 65535} {
		a := announcement{ID: "other-" + strconv.Itoa(port), Name: "Other", Port: port}
		raw := a.ID + "|" + a.Name + "|" + strconv.Itoa(a.Port)
		a.Signature = core.Sign("long-enough-key", []byte(raw))
		payload, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}

		n.recordAnnouncement(append([]byte(discoveryPrefix), payload...), &net.UDPAddr{IP: net.ParseIP("192.0.2.12"), Port: 8081})

		var address string
		for _, device := range n.Devices() {
			if device.ID == a.ID {
				address = device.Address
				break
			}
		}
		if got, want := address, net.JoinHostPort("192.0.2.12", strconv.Itoa(port)); got != want {
			t.Errorf("announcement port %d produced address %q, want %q", port, got, want)
		}
	}
}

func TestRecordAnnouncementRejectsInvalidSignature(t *testing.T) {
	n, err := New(Config{ID: "self", Name: "Self", Secret: "long-enough-key", Port: 8080, DiscoveryPort: 8081}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := announcement{ID: "other", Name: "Other", Port: 43210, Signature: "invalid"}
	payload, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}

	n.recordAnnouncement(append([]byte(discoveryPrefix), payload...), &net.UDPAddr{IP: net.ParseIP("192.0.2.12"), Port: 8081})
	if got := n.Devices(); len(got) != 0 {
		t.Fatalf("recorded unauthenticated device: %#v", got)
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "expected handler failure" }
