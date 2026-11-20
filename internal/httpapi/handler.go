// Package httpapi exposes the shortener over a small JSON REST API.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/asad110abbas/url-shortener-go/internal/shortener"
)

type createRequest struct {
	URL        string `json:"url"`
	CustomCode string `json:"custom_code,omitempty"`
	TTLHours   int    `json:"ttl_hours,omitempty"`
}

type createResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
	shortener.Link
}

// New returns the router. baseURL is used to build short links in responses.
func New(store *shortener.Store, baseURL string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/links", func(w http.ResponseWriter, r *http.Request) {
		var req createRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		link, err := store.Create(strings.TrimSpace(req.URL), req.CustomCode, time.Duration(req.TTLHours)*time.Hour)
		if err != nil {
			writeError(w, statusFor(err), err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, createResponse{Code: link.Code, ShortURL: strings.TrimRight(baseURL, "/") + "/" + link.Code, Link: *link})
	})
	mux.HandleFunc("GET /api/links/{code}", func(w http.ResponseWriter, r *http.Request) {
		link, err := store.Get(r.PathValue("code"))
		if err != nil {
			writeError(w, statusFor(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, link)
	})
	mux.HandleFunc("DELETE /api/links/{code}", func(w http.ResponseWriter, r *http.Request) {
		if err := store.Delete(r.PathValue("code")); err != nil {
			writeError(w, statusFor(err), err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /{code}", func(w http.ResponseWriter, r *http.Request) {
		target, err := store.Resolve(r.PathValue("code"))
		if err != nil {
			writeError(w, statusFor(err), err.Error())
			return
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
	return logging(mux)
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, shortener.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, shortener.ErrExpired):
		return http.StatusGone
	case errors.Is(err, shortener.ErrCodeTaken):
		return http.StatusConflict
	case errors.Is(err, shortener.ErrInvalidURL), errors.Is(err, shortener.ErrBadCode):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", time.Since(start))
	})
}
