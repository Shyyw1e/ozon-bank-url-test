package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Shyyw1e/ozon-bank-url-test/internal/core"
)

type Store struct {
	db *sql.DB
}

func New(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	var rel sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT to_regclass('public.url_mappings')`,
	).Scan(&rel); err != nil {
		_ = db.Close()
		return nil, err
	}
	if !rel.Valid {
		if _, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS public.url_mappings (
				code       VARCHAR(10) PRIMARY KEY,
				original   TEXT NOT NULL UNIQUE,
				created_at TIMESTAMPTZ NOT NULL DEFAULT now()
			)`); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) GetByOriginal(ctx context.Context, original string) (string, bool, error) {
	var code string
	err := s.db.QueryRowContext(ctx,
    	`SELECT code FROM public.url_mappings WHERE original = $1`, original,
	).Scan(&code)
	switch {
	case err == nil:
		return code, true, nil
	case errors.Is(err, sql.ErrNoRows):
		return "", false, nil
	default:
		return "", false, err
	}
}

func (s *Store) GetByCode(ctx context.Context, code string) (string, bool, error) {
	var original string
	err := s.db.QueryRowContext(ctx,
    	`SELECT original FROM public.url_mappings WHERE code = $1`, code,
	).Scan(&original)
	switch {
	case err == nil:
		return original, true, nil
	case errors.Is(err, sql.ErrNoRows):
		return "", false, nil
	default:
		return "", false, err
	}
}

func (s *Store) Create(ctx context.Context, code, original string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO public.url_mappings(code, original) VALUES ($1, $2)`,
		code, original,
	)
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { 
		switch pgErr.ConstraintName {
		case "url_mappings_pkey":
			return core.ErrDupCode
		case "url_mappings_original_key":
			return core.ErrDupOrigin
		}
	}
	return err
}
