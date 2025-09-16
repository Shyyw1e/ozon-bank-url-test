package httptransport

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	// Пишем JSON в буфер, чтобы потом проверить наличие полей
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	}))
}

func TestRouter_HealthzAndReadyz(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	h := NewRouter(log)

	// /healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("/healthz code=%d want=%d", rr.Code, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Fatalf("/healthz body=%q", rr.Body.String())
	}

	// убедимся, что лог миддлвари что-то записал
	if got := buf.String(); !strings.Contains(got, "http_request") || !strings.Contains(got, "\"status\":200") {
		t.Fatalf("log not contains expected attrs, got: %s", got)
	}

	// /readyz
	buf.Reset()
	req2 := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("/readyz code=%d want=%d", rr2.Code, http.StatusOK)
	}
	if !strings.Contains(rr2.Body.String(), "ready") {
		t.Fatalf("/readyz body=%q", rr2.Body.String())
	}
	if got := buf.String(); !strings.Contains(got, "http_request") || !strings.Contains(got, "\"status\":200") {
		t.Fatalf("log not contains expected attrs (readyz), got: %s", got)
	}
}

func TestRouter_Metrics(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)
	h := NewRouter(log)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("/metrics code=%d want=%d", rr.Code, http.StatusOK)
	}
	// promtext формат начинается с # HELP/TYPE и text/plain; version=0.0.4
	ct := rr.Result().Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("/metrics content-type=%q, want text/plain", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "# HELP") || !strings.Contains(body, "go_goroutines") {
		t.Fatalf("/metrics unexpected body (truncated): %.200s", body)
	}
}

func TestLoggingMiddleware_IncludesBasicAttrs(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	// Заворачиваем заглушку хэндлера в логирующую миддлварь
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond) // чтобы duration_ms был не нулевым
		w.WriteHeader(http.StatusTeapot) // 418
		_, _ = w.Write([]byte("teapot"))
	})

	mw := LoggingMiddleware(log)(next)

	req := httptest.NewRequest(http.MethodGet, "/brew/coffee", nil)
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status=%d want=418", rr.Code)
	}

	logOut := buf.String()
	for _, sub := range []string{
		"http_request",
		"\"method\":\"GET\"",
		"\"path\":\"/brew/coffee\"",
		"\"status\":418",
		"\"duration_ms\":",
		"\"request_id\":", // выставляется chi/middleware.RequestID в роутере, но здесь может быть пусто,
	} {
		if !strings.Contains(logOut, sub) {
			t.Fatalf("log missing %s; got: %s", sub, logOut)
		}
	}
}
