package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"lan-clipboard-sync/internal/core"
)

const discoveryPrefix = "LANSYNC1 "

type Config struct {
	ID, Name, Secret string
	Port             int
	DiscoveryPort    int
}
type Node struct {
	cfg       Config
	devices   *core.Devices
	seen      *core.Seen
	onMessage func(core.Message) error
	server    *http.Server
	mu        sync.Mutex
}
type announcement struct {
	ID, Name  string
	Port      int
	Signature string
}

func New(cfg Config, onMessage func(core.Message) error) (*Node, error) {
	if strings.TrimSpace(cfg.ID) == "" || strings.TrimSpace(cfg.Name) == "" || len(cfg.Secret) < 8 {
		return nil, errors.New("device id/name and an 8+ character shared key are required")
	}
	if cfg.Port < 1 || cfg.Port > 65535 || cfg.DiscoveryPort < 1 || cfg.DiscoveryPort > 65535 {
		return nil, errors.New("ports must be between 1 and 65535")
	}
	return &Node{cfg: cfg, devices: core.NewDevices(45 * time.Second), seen: core.NewSeen(24 * time.Hour), onMessage: onMessage}, nil
}
func (n *Node) Devices() []core.Device { return n.devices.List(time.Now()) }
func (n *Node) Port() int              { return n.cfg.Port }
func (n *Node) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clipboard", n.receive)
	n.server = &http.Server{Addr: net.JoinHostPort("", strconv.Itoa(n.cfg.Port)), Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	udp, err := net.ListenUDP("udp4", &net.UDPAddr{Port: n.cfg.DiscoveryPort})
	if err != nil {
		return fmt.Errorf("listen for discovery: %w", err)
	}
	defer udp.Close()
	go n.discover(ctx, udp)
	go n.announce(ctx)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = n.server.Shutdown(shutdown)
	}()
	err = n.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
func (n *Node) announce(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		n.sendAnnouncement()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (n *Node) sendAnnouncement() {
	a := announcement{ID: n.cfg.ID, Name: n.cfg.Name, Port: n.cfg.Port}
	raw := a.ID + "|" + a.Name + "|" + strconv.Itoa(a.Port)
	a.Signature = core.Sign(n.cfg.Secret, []byte(raw))
	b, err := json.Marshal(a)
	if err != nil {
		return
	}
	c, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4bcast, Port: n.cfg.DiscoveryPort})
	if err != nil {
		return
	}
	defer c.Close()
	_, _ = c.Write([]byte(discoveryPrefix + string(b)))
}
func (n *Node) discover(ctx context.Context, c *net.UDPConn) {
	buf := make([]byte, 2048)
	for {
		_ = c.SetReadDeadline(time.Now().Add(time.Second))
		size, from, err := c.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			continue
		}
		n.recordAnnouncement(buf[:size], from)
	}
}
func (n *Node) recordAnnouncement(data []byte, from *net.UDPAddr) {
	if !strings.HasPrefix(string(data), discoveryPrefix) {
		return
	}
	var a announcement
	if json.Unmarshal(data[len(discoveryPrefix):], &a) != nil || a.ID == n.cfg.ID || a.Port < 1 || a.Port > 65535 || a.Name == "" {
		return
	}
	raw := a.ID + "|" + a.Name + "|" + strconv.Itoa(a.Port)
	if !core.ValidSignature(n.cfg.Secret, []byte(raw), a.Signature) {
		return
	}
	n.devices.Upsert(core.Device{ID: a.ID, Name: a.Name, Address: net.JoinHostPort(from.IP.String(), strconv.Itoa(n.cfg.DiscoveryPort)), LastSeen: time.Now()})
}
func (n *Node) receive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, core.MaxTextBytes+4096)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !core.ValidSignature(n.cfg.Secret, payload, r.Header.Get("X-LanSync-Signature")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	m, err := core.Decode(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !n.seen.AddIfNew(m.ID, time.Now()) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if n.onMessage != nil {
		if err := n.onMessage(m); err != nil {
			n.seen.Remove(m.ID)
			http.Error(w, "could not accept clipboard text", http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
func (n *Node) Send(ctx context.Context, address string, text string) error {
	m := core.Message{ID: newID(), SenderID: n.cfg.ID, Sender: n.cfg.Name, Text: text, Created: time.Now().UTC()}
	if err := m.Validate(); err != nil {
		return err
	}
	payload, err := core.Encode(m)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+address+"/v1/clipboard", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-LanSync-Signature", core.Sign(n.cfg.Secret, payload))
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("receiver returned %s", resp.Status)
	}
	return nil
}
func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
