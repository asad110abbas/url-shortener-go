package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/asad110abbas/url-shortener-go/internal/shortener"
)

func newServer(t *testing.T) *httptest.Server {
	store, _ := shortener.NewStore("")
	srv := httptest.NewServer(New(store, "http://sho.rt"))
	t.Cleanup(srv.Close)
	return srv
}

func TestShortenRedirectStats(t *testing.T) {
	srv := newServer(t)
	res, err := http.Post(srv.URL+"/api/links", "application/json",
		strings.NewReader(`{"url":"https://example.com/docs","custom_code":"docs"}`))
	if err != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("create: %v %d", err, res.StatusCode)
	}
	var body createResponse
	_ = json.NewDecoder(res.Body).Decode(&body)
	if body.ShortURL != "http://sho.rt/docs" {
		t.Fatalf("short url = %q", body.ShortURL)
	}

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r, _ := client.Get(srv.URL + "/docs")
	if r.StatusCode != http.StatusFound || r.Header.Get("Location") != "https://example.com/docs" {
		t.Fatalf("redirect: %d %q", r.StatusCode, r.Header.Get("Location"))
	}

	s, _ := http.Get(srv.URL + "/api/links/docs")
	var link shortener.Link
	_ = json.NewDecoder(s.Body).Decode(&link)
	if link.Clicks != 1 {
		t.Fatalf("clicks = %d", link.Clicks)
	}
}

func TestErrorsMapToStatusCodes(t *testing.T) {
	srv := newServer(t)
	cases := []struct {
		body string
		want int
	}{
		{`{"url":"nope"}`, http.StatusBadRequest},
		{`not json`, http.StatusBadRequest},
	}
	for _, c := range cases {
		res, _ := http.Post(srv.URL+"/api/links", "application/json", strings.NewReader(c.body))
		if res.StatusCode != c.want {
			t.Errorf("%s -> %d, want %d", c.body, res.StatusCode, c.want)
		}
	}
	if res, _ := http.Get(srv.URL + "/missing"); res.StatusCode != http.StatusNotFound {
		t.Errorf("missing code -> %d", res.StatusCode)
	}
}
