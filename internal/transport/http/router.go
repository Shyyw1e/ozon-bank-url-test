package httptransport

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/Shyyw1e/ozon-bank-url-test/internal/core"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)


func NewRouter(log *slog.Logger, svc *core.Shortener) http.Handler{
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10*time.Second))

	r.Use(LoggingMiddleware(log))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})
	r.Post("/api/v1/urls", func(w http.ResponseWriter, r *http.Request) {
		type RequestPOST struct {
			URL string `json:"url"`
		}
		type ResponsePOST struct {
			Code     string `json:"code"`
			ShortURL string `json:"short_url"`
		}

		var req RequestPOST
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("invalid request json", "err", err)
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		code, err := svc.Create(r.Context(), req.URL)
		if err != nil {
			switch err {
			case core.ErrInvalidURL:
				http.Error(w, "invalid url", http.StatusBadRequest)
			case core.ErrConflict:
				http.Error(w, "too many collisions", http.StatusConflict)
			default:
				log.Error("create failed", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		short := absoluteURL(r, code)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", short)
		w.WriteHeader(http.StatusCreated)

		_ = json.NewEncoder(w).Encode(ResponsePOST{
			Code:     code,
			ShortURL: short,
		})
	})

	r.Get("/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		if !core.IsValidCode(code) {
			log.Error("invalid code")
			http.NotFound(w, r)
			return 
		}

		original, err := svc.Resolve(r.Context(), code)
		switch err {
		case core.ErrNotFound:
			http.NotFound(w, r)
			return 
		case nil:
			http.Redirect(w, r, original, http.StatusFound)
		default:
			log.Error("resolve failed", "code", code, "err", err)
            http.Error(w, "internal error", http.StatusInternalServerError)
			return 
		}

	})

	r.Get("/api/v1/urls/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")

		original, err := svc.Resolve(r.Context(), code)
		if err != nil {
			if err == core.ErrNotFound {
				http.NotFound(w, r)
				return
			}
			log.Error("resolve failed", "code", code, "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			URL string `json:"url"`
		}{URL: original})
	})


	r.Handle("/metrics", promhttp.Handler())

	return r
}

func absoluteURL(r *http.Request, code string) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		proto = "http"
	}
	return proto + "://" + r.Host + "/" + code
}
