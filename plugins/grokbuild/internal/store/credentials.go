package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Credential is a SuperGrok / Grok CLI credential with pool health fields.
type Credential struct {
	ID           string     `json:"id"`
	Label        string     `json:"label"`
	Email        string     `json:"email,omitempty"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at,omitempty"`
	Enabled      bool       `json:"enabled"`
	Priority     int        `json:"priority"`
	FailureCount int        `json:"failure_count,omitempty"`
	CooldownUntil *time.Time `json:"cooldown_until,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	LastSuccess  *time.Time `json:"last_success_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type fileData struct {
	Credentials []Credential `json:"credentials"`
	RRIndex     int          `json:"rr_index,omitempty"`
}

// Store is a process-local JSON credential store under the plugin data dir.
type Store struct {
	path string
	mu   sync.Mutex
}

func New(dataDir string) *Store {
	return &Store{path: filepath.Join(dataDir, "credentials.json")}
}

func (s *Store) List() ([]Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]Credential, len(data.Credentials))
	copy(out, data.Credentials)
	sortCredentials(out)
	return out, nil
}

// Pick selects the next usable credential (priority desc, round-robin among equals).
func (s *Store) Pick() (Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return Credential{}, err
	}
	now := time.Now().UTC()
	usable := make([]Credential, 0, len(data.Credentials))
	for _, c := range data.Credentials {
		if !c.Enabled || strings.TrimSpace(c.AccessToken) == "" {
			continue
		}
		if c.CooldownUntil != nil && c.CooldownUntil.After(now) {
			continue
		}
		usable = append(usable, c)
	}
	if len(usable) == 0 {
		return Credential{}, errors.New("no enabled grok credential; open the GrokBuild plugin page to add a token")
	}
	sortCredentials(usable)
	// Round-robin within the full usable set, preferring higher priority first
	// by rotating only among the top priority band when possible.
	top := usable[0].Priority
	band := make([]Credential, 0, len(usable))
	for _, c := range usable {
		if c.Priority == top {
			band = append(band, c)
		}
	}
	if len(band) == 0 {
		band = usable
	}
	if data.RRIndex < 0 {
		data.RRIndex = 0
	}
	idx := data.RRIndex % len(band)
	data.RRIndex = (data.RRIndex + 1) % len(band)
	_ = s.saveLocked(data)
	return band[idx], nil
}

// ListUsableIDs returns enabled non-cooling credential IDs for failover retries.
func (s *Store) ListUsable() ([]Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]Credential, 0, len(data.Credentials))
	for _, c := range data.Credentials {
		if !c.Enabled || strings.TrimSpace(c.AccessToken) == "" {
			continue
		}
		if c.CooldownUntil != nil && c.CooldownUntil.After(now) {
			continue
		}
		out = append(out, c)
	}
	sortCredentials(out)
	return out, nil
}

func (s *Store) MarkSuccess(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i := range data.Credentials {
		if data.Credentials[i].ID != id {
			continue
		}
		data.Credentials[i].FailureCount = 0
		data.Credentials[i].CooldownUntil = nil
		data.Credentials[i].LastError = ""
		data.Credentials[i].LastUsedAt = &now
		data.Credentials[i].LastSuccess = &now
		data.Credentials[i].UpdatedAt = now
		return s.saveLocked(data)
	}
	return errors.New("credential not found")
}

// MarkFailure records a failure. status 429 uses Retry-After when >0 seconds.
func (s *Store) MarkFailure(id string, status int, retryAfterSec int, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i := range data.Credentials {
		if data.Credentials[i].ID != id {
			continue
		}
		c := &data.Credentials[i]
		c.FailureCount++
		c.LastError = truncate(message, 240)
		c.LastUsedAt = &now
		c.UpdatedAt = now
		cd := cooldownFor(status, retryAfterSec, c.FailureCount)
		if cd > 0 {
			until := now.Add(cd)
			c.CooldownUntil = &until
		}
		// Disable after repeated hard auth failures.
		if status == 401 || status == 403 {
			if c.FailureCount >= 3 {
				c.Enabled = false
				c.LastError = "disabled after repeated auth failures: " + c.LastError
			}
		}
		return s.saveLocked(data)
	}
	return errors.New("credential not found")
}

func (s *Store) Upsert(c Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	c.UpdatedAt = now
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.ID == "" {
		c.ID = "cred-" + now.Format("20060102150405")
	}
	found := false
	for i := range data.Credentials {
		if data.Credentials[i].ID == c.ID {
			// Preserve health counters unless explicitly reset via fields zeroed by caller.
			prev := data.Credentials[i]
			if c.AccessToken == "" {
				c.AccessToken = prev.AccessToken
			}
			if c.RefreshToken == "" {
				c.RefreshToken = prev.RefreshToken
			}
			c.FailureCount = prev.FailureCount
			c.CooldownUntil = prev.CooldownUntil
			c.LastError = prev.LastError
			c.LastUsedAt = prev.LastUsedAt
			c.LastSuccess = prev.LastSuccess
			if c.CreatedAt.IsZero() {
				c.CreatedAt = prev.CreatedAt
			}
			data.Credentials[i] = c
			found = true
			break
		}
	}
	if !found {
		data.Credentials = append(data.Credentials, c)
	}
	return s.saveLocked(data)
}

// ApplyTokens persists refreshed access/refresh tokens for a credential.
func (s *Store) ApplyTokens(id, access, refresh string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i := range data.Credentials {
		if data.Credentials[i].ID != id {
			continue
		}
		if strings.TrimSpace(access) != "" {
			data.Credentials[i].AccessToken = strings.TrimSpace(access)
		}
		if strings.TrimSpace(refresh) != "" {
			data.Credentials[i].RefreshToken = strings.TrimSpace(refresh)
		}
		if !expiresAt.IsZero() {
			data.Credentials[i].ExpiresAt = expiresAt
		}
		data.Credentials[i].UpdatedAt = now
		return s.saveLocked(data)
	}
	return errors.New("credential not found")
}

// Get returns one credential by id.
func (s *Store) Get(id string) (Credential, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return Credential{}, err
	}
	for _, c := range data.Credentials {
		if c.ID == id {
			return c, nil
		}
	}
	return Credential{}, errors.New("credential not found")
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadLocked()
	if err != nil {
		return err
	}
	next := data.Credentials[:0]
	for _, c := range data.Credentials {
		if c.ID != id {
			next = append(next, c)
		}
	}
	data.Credentials = next
	return s.saveLocked(data)
}

// ImportRaw parses common Grok auth JSON shapes and upserts credentials.
func (s *Store) ImportRaw(raw []byte, defaultLabel string) (int, error) {
	items, err := ParseImport(raw, defaultLabel)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, item := range items {
		if err := s.Upsert(item); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (s *Store) loadLocked() (fileData, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fileData{}, nil
		}
		return fileData{}, err
	}
	var data fileData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fileData{}, err
	}
	return data, nil
}

func (s *Store) saveLocked(data fileData) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func sortCredentials(items []Credential) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		return items[i].ID < items[j].ID
	})
}

func cooldownFor(status, retryAfterSec, failures int) time.Duration {
	if retryAfterSec > 0 {
		d := time.Duration(retryAfterSec) * time.Second
		if d > 15*time.Minute {
			d = 15 * time.Minute
		}
		return d
	}
	switch status {
	case 429:
		// exponential-ish backoff
		sec := 30 << min(failures-1, 4) // 30,60,120,240,480
		return time.Duration(sec) * time.Second
	case 401, 403:
		return 2 * time.Minute
	case 402:
		return 10 * time.Minute
	case 500, 502, 503, 504:
		return 30 * time.Second
	default:
		if failures >= 2 {
			return 15 * time.Second
		}
		return 0
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
