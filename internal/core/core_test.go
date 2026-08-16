package core

import (
	"testing"
	"time"
)

func TestSignature(t *testing.T) {
	payload := []byte(`{"text":"hello"}`)
	sig := Sign("key", payload)
	if !ValidSignature("key", payload, sig) || ValidSignature("wrong", payload, sig) {
		t.Fatal("signature validation failed")
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

func TestHistoryKeepsNewestEntriesFirst(t *testing.T) {
	h := NewHistory(3)
	for _, id := range []string{"one", "two", "three"} {
		h.Add(Message{ID: id})
	}

	entries := h.List()
	for i, want := range []string{"three", "two", "one"} {
		if entries[i].ID != want {
			t.Fatalf("entry %d = %q, want %q", i, entries[i].ID, want)
		}
	}
}

func TestHistoryDiscardsOldestEntryAtCapacity(t *testing.T) {
	h := NewHistory(2)
	for _, id := range []string{"one", "two", "three"} {
		h.Add(Message{ID: id})
	}

	entries := h.List()
	if len(entries) != 2 || entries[0].ID != "three" || entries[1].ID != "two" {
		t.Fatalf("entries = %#v, want newest two entries first", entries)
	}
}
