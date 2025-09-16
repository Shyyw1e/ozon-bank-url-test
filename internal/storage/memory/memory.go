// internal/storage/memory/memory.go
package memory

import (
	"context"
	"sync"

	"github.com/Shyyw1e/ozon-bank-url-test/internal/core"
)

type Store struct {
    mu     sync.RWMutex
    byOrig map[string]string // original -> code
    byCode map[string]string // code -> original
}

func New() *Store {
    return &Store{
        byOrig: make(map[string]string),
        byCode: make(map[string]string),
    }
}

func (s *Store) GetByOriginal(ctx context.Context, original string) (string, bool, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    code, ok := s.byOrig[original]
    return code, ok, nil
}

func (s *Store) GetByCode(ctx context.Context, code string) (string, bool, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    original, ok := s.byCode[code]
    return original, ok, nil
}

func (s *Store) Create(ctx context.Context, code, original string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, ok := s.byOrig[original]; ok {
        return core.ErrDupOrigin
    }
    if _, ok := s.byCode[code]; ok {
        return core.ErrDupCode
    }
    s.byOrig[original] = code
    s.byCode[code] = original
    return nil
}
