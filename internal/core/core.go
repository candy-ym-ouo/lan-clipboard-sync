package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const MaxTextBytes = 1 << 20

type Message struct {
	ID       string    `json:"id"`
	SenderID string    `json:"sender_id"`
	Sender   string    `json:"sender"`
	Text     string    `json:"text"`
	Created  time.Time `json:"created"`
}

func (m Message) Validate() error {
	if strings.TrimSpace(m.ID) == "" || strings.TrimSpace(m.SenderID) == "" || strings.TrimSpace(m.Sender) == "" {
		return errors.New("message id, sender id, and sender are required")
	}
	if len(m.Text) == 0 {
		return errors.New("clipboard text cannot be empty")
	}
	if len(m.Text) > MaxTextBytes {
		return fmt.Errorf("clipboard text exceeds %d bytes", MaxTextBytes)
	}
	if m.Created.IsZero() {
		return errors.New("message creation time is required")
	}
	return nil
}

func Sign(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	_, _ = h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func ValidSignature(secret string, payload []byte, signature string) bool {
	expected := Sign(secret, payload)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

type Seen struct {
	mu  sync.Mutex
	ids map[string]time.Time
	ttl time.Duration
}

func NewSeen(ttl time.Duration) *Seen { return &Seen{ids: make(map[string]time.Time), ttl: ttl} }
func (s *Seen) AddIfNew(id string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, expiry := range s.ids {
		if !expiry.After(now) {
			delete(s.ids, key)
		}
	}
	if _, found := s.ids[id]; found {
		return false
	}
	s.ids[id] = now.Add(s.ttl)
	return true
}

// Remove allows a failed receiver operation to be retried with the same message ID.
func (s *Seen) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ids, id)
}

type Device struct {
	ID, Name, Address string
	LastSeen          time.Time
}
type Devices struct {
	mu     sync.RWMutex
	values map[string]Device
	ttl    time.Duration
}

func NewDevices(ttl time.Duration) *Devices {
	return &Devices{values: make(map[string]Device), ttl: ttl}
}
func (d *Devices) Upsert(device Device) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.values[device.ID] = device
}
func (d *Devices) List(now time.Time) []Device {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]Device, 0, len(d.values))
	for id, device := range d.values {
		if now.Sub(device.LastSeen) > d.ttl {
			delete(d.values, id)
			continue
		}
		out = append(out, device)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

type History struct {
	mu      sync.Mutex
	entries []Message
	max     int
}

func NewHistory(max int) *History {
	if max < 1 {
		max = 1
	}
	return &History{max: max}
}
func (h *History) Add(m Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, m)
	if len(h.entries) > h.max {
		h.entries = h.entries[:h.max]
	}
}
func (h *History) List() []Message {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Message(nil), h.entries...)
}
func Encode(m Message) ([]byte, error) { return json.Marshal(m) }
func Decode(b []byte) (Message, error) {
	var m Message
	err := json.Unmarshal(b, &m)
	if err == nil {
		err = m.Validate()
	}
	return m, err
}
