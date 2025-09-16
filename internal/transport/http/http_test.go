package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shyyw1e/ozon-bank-url-test/internal/core"
	"github.com/Shyyw1e/ozon-bank-url-test/internal/storage/memory"
)

func testLogger() *slog.Logger {
	var buf bytes.Buffer
	return slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	}))
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	st := memory.New()
	svc := core.NewShortener(st, core.NewCode)
	return NewRouter(testLogger(), svc)
}

func TestPOST_Create_OK(t *testing.T) {
	h := newTestRouter(t)

	body := `{"url":"https://example.com/path?q=1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d, want=%d", rr.Code, http.StatusCreated)
	}
	var resp struct {
		Code     string `json:"code"`
		ShortURL string `json:"short_url"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v; body=%q", err, rr.Body.String())
	}
	if !core.IsValidCode(resp.Code) {
		t.Fatalf("invalid code in response: %q", resp.Code)
	}
	if !strings.HasPrefix(resp.ShortURL, "https://") {
		t.Fatalf("short_url must start with https://, got %q", resp.ShortURL)
	}
	if loc := rr.Header().Get("Location"); loc != resp.ShortURL {
		t.Fatalf("Location header mismatch: %q vs %q", loc, resp.ShortURL)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type=%q, want application/json", ct)
	}
}

func TestPOST_Create_InvalidURL(t *testing.T) {
	h := newTestRouter(t)

	body := `{"url":"ftp://bad.example"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rr.Code)
	}
}

func TestPOST_Create_Idempotent_SameCode(t *testing.T) {
	h := newTestRouter(t)
	url := `{"url":"https://example.com/idem"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/urls", strings.NewReader(url))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("first create status=%d", rr1.Code)
	}
	var r1 struct {
		Code     string `json:"code"`
		ShortURL string `json:"short_url"`
	}
	_ = json.Unmarshal(rr1.Body.Bytes(), &r1)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/urls", strings.NewReader(url))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated && rr2.Code != http.StatusOK {
		t.Fatalf("second create status=%d, want 201 or 200", rr2.Code)
	}
	var r2 struct {
		Code     string `json:"code"`
		ShortURL string `json:"short_url"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &r2)

	if r1.Code != r2.Code {
		t.Fatalf("idempotency violated: %q vs %q", r1.Code, r2.Code)
	}
}

func TestGET_Code_Redirect_Found(t *testing.T) {
	// Подготовим запись через сервис, затем проверим редирект
	st := memory.New()
	svc := core.NewShortener(st, core.NewCode)
	orig := "https://golang.org"
	// создадим заранее
	code, err := svc.Create(context.Background(), orig)
	if err != nil {
		t.Fatalf("prep Create err: %v", err)
	}
	h := NewRouter(testLogger(), svc)

	req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status=%d, want 302", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != orig {
		t.Fatalf("Location=%q, want %q", loc, orig)
	}
}

func TestGET_Code_NotFound(t *testing.T) {
	h := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/NO_SUCH__1", nil) // 10 символов, валидный формат, но не в store
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rr.Code)
	}
}

func TestGET_JSON_Resolve_OK(t *testing.T) {
	st := memory.New()
	svc := core.NewShortener(st, core.NewCode)
	orig := "https://example.com/json"
	code, err := svc.Create(context.Background(), orig)
	if err != nil {
		t.Fatalf("prep Create err: %v", err)
	}
	h := NewRouter(testLogger(), svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/urls/"+code, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rr.Code)
	}
	var resp struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp.URL != orig {
		t.Fatalf("url=%q, want %q", resp.URL, orig)
	}
}
