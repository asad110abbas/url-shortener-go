package shortener

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCreateAndResolve(t *testing.T) {
	s, _ := NewStore("")
	link, err := s.Create("https://example.com/a/very/long/path", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(link.Code) != 7 {
		t.Fatalf("expected 7-char code, got %q", link.Code)
	}
	target, err := s.Resolve(link.Code)
	if err != nil || target != "https://example.com/a/very/long/path" {
		t.Fatalf("resolve: %v %q", err, target)
	}
	got, _ := s.Get(link.Code)
	if got.Clicks != 1 || got.LastClick == nil {
		t.Fatalf("click not recorded: %+v", got)
	}
}

func TestValidation(t *testing.T) {
	s, _ := NewStore("")
	cases := []struct {
		url, code string
		want      error
	}{
		{"not a url", "", ErrInvalidURL},
		{"ftp://example.com", "", ErrInvalidURL},
		{"https://example.com", "a!", ErrBadCode},
	}
	for _, c := range cases {
		if _, err := s.Create(c.url, c.code, 0); !errors.Is(err, c.want) {
			t.Errorf("Create(%q,%q) = %v, want %v", c.url, c.code, err, c.want)
		}
	}
}

func TestCustomCodeConflict(t *testing.T) {
	s, _ := NewStore("")
	if _, err := s.Create("https://example.com", "promo", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("https://example.org", "promo", 0); !errors.Is(err, ErrCodeTaken) {
		t.Fatalf("expected ErrCodeTaken, got %v", err)
	}
}

func TestExpiry(t *testing.T) {
	s, _ := NewStore("")
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return base }
	link, _ := s.Create("https://example.com", "", time.Hour)
	s.now = func() time.Time { return base.Add(2 * time.Hour) }
	if _, err := s.Resolve(link.Code); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.json")
	s, _ := NewStore(path)
	link, _ := s.Create("https://example.com", "keep", 0)
	s2, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Get(link.Code); err != nil {
		t.Fatalf("link not persisted: %v", err)
	}
}

func TestConcurrentResolve(t *testing.T) {
	s, _ := NewStore("")
	link, _ := s.Create("https://example.com", "", 0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = s.Resolve(link.Code) }()
	}
	wg.Wait()
	if got, _ := s.Get(link.Code); got.Clicks != 100 {
		t.Fatalf("expected 100 clicks, got %d", got.Clicks)
	}
}
