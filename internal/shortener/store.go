// Package shortener contains the core URL shortening logic, independent of HTTP.
package shortener

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"
)

const alphabet = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no look-alikes (0/O, 1/l/I)

var (
	ErrNotFound   = errors.New("short code not found")
	ErrExpired    = errors.New("link has expired")
	ErrInvalidURL = errors.New("url must be absolute http(s)")
	ErrCodeTaken  = errors.New("custom code already in use")
	ErrBadCode    = errors.New("custom code must be 3-32 chars: letters, digits, - or _")
	codePattern   = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)
)

// Link is one shortened URL with its usage stats.
type Link struct {
	Code      string     `json:"code"`
	Target    string     `json:"target"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Clicks    int64      `json:"clicks"`
	LastClick *time.Time `json:"last_click,omitempty"`
}

// Store is a concurrency-safe in-memory store with optional JSON-file persistence.
type Store struct {
	mu    sync.RWMutex
	links map[string]*Link
	path  string
	now   func() time.Time
}

// NewStore creates a store. If path is non-empty, links are loaded from and saved to that file.
func NewStore(path string) (*Store, error) {
	s := &Store{links: map[string]*Link{}, path: path, now: time.Now}
	if path == "" {
		return s, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	return s, json.Unmarshal(data, &s.links)
}

func validURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func randomCode(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alphabet[idx.Int64()]
	}
	return string(b), nil
}

// Create shortens target. custom may be empty; ttl of 0 means no expiry.
func (s *Store) Create(target, custom string, ttl time.Duration) (*Link, error) {
	if !validURL(target) {
		return nil, ErrInvalidURL
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	code := custom
	if code != "" {
		if !codePattern.MatchString(code) {
			return nil, ErrBadCode
		}
		if _, taken := s.links[code]; taken {
			return nil, ErrCodeTaken
		}
	} else {
		for {
			c, err := randomCode(7)
			if err != nil {
				return nil, err
			}
			if _, taken := s.links[c]; !taken {
				code = c
				break
			}
		}
	}
	link := &Link{Code: code, Target: target, CreatedAt: s.now().UTC()}
	if ttl > 0 {
		exp := link.CreatedAt.Add(ttl)
		link.ExpiresAt = &exp
	}
	s.links[code] = link
	return link, s.saveLocked()
}

// Resolve returns the target for code and records a click.
func (s *Store) Resolve(code string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.links[code]
	if !ok {
		return "", ErrNotFound
	}
	now := s.now().UTC()
	if link.ExpiresAt != nil && now.After(*link.ExpiresAt) {
		return "", ErrExpired
	}
	link.Clicks++
	link.LastClick = &now
	return link.Target, s.saveLocked()
}

// Get returns a copy of the link and its stats.
func (s *Store) Get(code string) (Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.links[code]
	if !ok {
		return Link{}, ErrNotFound
	}
	return *link, nil
}

// Delete removes a link.
func (s *Store) Delete(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.links[code]; !ok {
		return ErrNotFound
	}
	delete(s.links, code)
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(s.links, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path) // atomic replace
}
