package core

import (
	"testing"
	"time"
)

func TestSignature(t *testing.T) {
	validPayload := []byte(`{"text":"hello"}`)
	validSignature := Sign("key", validPayload)
	tests := []struct {
		name      string
		secret    string
		payload   []byte
		signature string
		valid     bool
	}{
		{name: "valid", secret: "key", payload: validPayload, signature: validSignature, valid: true},
		{name: "empty secret and payload", signature: Sign("", nil), valid: true},
		{name: "wrong secret", secret: "wrong", payload: validPayload, signature: validSignature},
		{name: "modified payload", secret: "key", payload: []byte(`{"text":"hellO"}`), signature: validSignature},
		{name: "empty signature", secret: "key", payload: []byte(`{"text":"hello"}`), signature: ""},
		{name: "short signature", secret: "key", payload: []byte(`{"text":"hello"}`), signature: "0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidSignature(test.secret, test.payload, test.signature); got != test.valid {
				t.Fatalf("ValidSignature() = %t, want %t", got, test.valid)
			}
		})
	}
}
func TestSeenExpires(t *testing.T) {
	now := time.Now()
	seen := NewSeen(time.Second)
	if !seen.AddIfNew("a", now) || seen.AddIfNew("a", now) || !seen.AddIfNew("a", now.Add(2*time.Second)) {
		t.Fatal("unexpected duplicate handling")
	}
}
func TestMessageValidation(t *testing.T) {
	m := Message{ID: "1", SenderID: "a", Sender: "A", Text: "ok", Created: time.Now()}
	b, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(b)
	if err != nil || got.Text != "ok" {
		t.Fatalf("decode: %v, %#v", err, got)
	}
	m.Text = ""
	if err := m.Validate(); err == nil {
		t.Fatal("empty text was accepted")
	}
}

func TestHistoryAcceptsInvalidLimit(t *testing.T) {
	h := NewHistory(-1)
	h.Add(Message{ID: "1"})
	if len(h.List()) != 1 {
		t.Fatal("history did not retain entry")
	}
}
