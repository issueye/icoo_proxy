package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// DeviceSession tracks an in-flight browser device login.
type DeviceSession struct {
	ID                      string    `json:"id"`
	UserCode                string    `json:"user_code"`
	VerificationURI         string    `json:"verification_uri"`
	VerificationURIComplete string    `json:"verification_uri_complete,omitempty"`
	Interval                int       `json:"interval"`
	ExpiresAt               time.Time `json:"expires_at"`
	Status                  string    `json:"status"` // pending | success | error | expired
	Error                   string    `json:"error,omitempty"`
	CredentialID            string    `json:"credential_id,omitempty"`

	deviceCode string
}

// SessionManager runs device login polls and exposes status to the admin UI.
type SessionManager struct {
	Client *Client
	OnSuccess func(sessionID string, tokens TokenSet) (credentialID string, err error)

	mu       sync.Mutex
	sessions map[string]*DeviceSession
}

func NewSessionManager(client *Client, onSuccess func(string, TokenSet) (string, error)) *SessionManager {
	if client == nil {
		client = NewClient()
	}
	return &SessionManager{
		Client:    client,
		OnSuccess: onSuccess,
		sessions:  make(map[string]*DeviceSession),
	}
}

// StartDeviceLogin begins RFC 8628 device flow and background polling.
func (m *SessionManager) StartDeviceLogin(ctx context.Context) (*DeviceSession, error) {
	_ = m.Client.Discover(ctx)
	dc, err := m.Client.RequestDeviceCode(ctx)
	if err != nil {
		return nil, err
	}
	id := newID("dev")
	exp := time.Now().UTC().Add(time.Duration(dc.ExpiresIn) * time.Second)
	if dc.ExpiresIn <= 0 {
		exp = time.Now().UTC().Add(15 * time.Minute)
	}
	sess := &DeviceSession{
		ID:                      id,
		UserCode:                dc.UserCode,
		VerificationURI:         dc.VerificationURI,
		VerificationURIComplete: dc.VerificationURIComplete,
		Interval:                dc.Interval,
		ExpiresAt:               exp,
		Status:                  "pending",
		deviceCode:              dc.DeviceCode,
	}
	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

	go m.poll(id, dc)
	return copySession(sess), nil
}

func (m *SessionManager) Get(id string) (*DeviceSession, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, false
	}
	return copySession(s), true
}

func (m *SessionManager) poll(id string, dc *DeviceCode) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Until(time.Now().Add(time.Duration(dc.ExpiresIn)*time.Second))+time.Minute)
	if dc.ExpiresIn <= 0 {
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), 16*time.Minute)
	}
	defer cancel()

	ts, err := m.Client.PollDevice(ctx, dc)
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	if err != nil {
		if ctx.Err() != nil || time.Now().After(sess.ExpiresAt) {
			sess.Status = "expired"
			sess.Error = "device login expired or canceled"
		} else {
			sess.Status = "error"
			sess.Error = err.Error()
		}
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	credID := ""
	if m.OnSuccess != nil {
		var persistErr error
		credID, persistErr = m.OnSuccess(id, *ts)
		m.mu.Lock()
		sess = m.sessions[id]
		if sess != nil {
			if persistErr != nil {
				sess.Status = "error"
				sess.Error = persistErr.Error()
			} else {
				sess.Status = "success"
				sess.CredentialID = credID
			}
		}
		m.mu.Unlock()
		return
	}
	m.mu.Lock()
	if sess = m.sessions[id]; sess != nil {
		sess.Status = "success"
	}
	m.mu.Unlock()
}

func copySession(s *DeviceSession) *DeviceSession {
	if s == nil {
		return nil
	}
	cp := *s
	cp.deviceCode = "" // never expose
	return &cp
}

func newID(prefix string) string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b[:]))
}
