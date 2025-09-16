package core

import (
	"context"
	"sync"
	"testing"
)


type fakeStore struct {
	mu     sync.Mutex
	byOrig map[string]string 
	byCode map[string]string 

	dupCodeLeft   int    
	forceDupOrig  bool  
	existingOrig  string 
	existingCode  string 
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		byOrig: make(map[string]string),
		byCode: make(map[string]string),
	}
}

func (s *fakeStore) GetByOriginal(ctx context.Context, original string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byOrig[original]
	return c, ok, nil
}

func (s *fakeStore) GetByCode(ctx context.Context, code string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.byCode[code]
	return o, ok, nil
}

func (s *fakeStore) Create(ctx context.Context, code, original string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.forceDupOrig && original == s.existingOrig {
		s.byOrig[s.existingOrig] = s.existingCode
		s.byCode[s.existingCode] = s.existingOrig
		return ErrDupOrigin
	}

	if s.dupCodeLeft > 0 {
		s.dupCodeLeft--
		return ErrDupCode
	}

	if _, ok := s.byOrig[original]; ok {
		return ErrDupOrigin
	}
	if _, ok := s.byCode[code]; ok {
		return ErrDupCode
	}

	s.byOrig[original] = code
	s.byCode[code] = original
	return nil
}

func stubGen(seq ...string) CodeGenerator {
	i := 0
	return func(n int) (string, error) {
		v := seq[i%len(seq)]
		i++
		return v, nil
	}
}


func TestCreate_IdempotentSameURL_ReturnsSameCode(t *testing.T) {
	store := newFakeStore()
	svc := NewShortener(store, stubGen("AAAAAAAAAA")) 

	u := "https://example.com/x"
	code1, err := svc.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create #1 err: %v", err)
	}
	code2, err := svc.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create #2 err: %v", err)
	}
	if code1 != code2 {
		t.Fatalf("expected same code, got %q vs %q", code1, code2)
	}
}

func TestCreate_RetryOnDupCode_Succeeds(t *testing.T) {
	store := newFakeStore()
	store.dupCodeLeft = 1 
	svc := NewShortener(store, stubGen("AAAAAAAAAA", "BBBBBBBBBB"))

	u := "https://example.com/y"
	code, err := svc.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create err: %v", err)
	}
	if code != "BBBBBBBBBB" {
		t.Fatalf("expected second code after retry, got %q", code)
	}
}

func TestCreate_TooManyCollisions_ReturnsErrConflict(t *testing.T) {
	store := newFakeStore()
	store.dupCodeLeft = 100 
	svc := NewShortener(store, stubGen("AAAAAAAAAA", "BBBBBBBBBB", "CCCCCCCCCC"))

	u := "https://example.com/z"
	_, err := svc.Create(context.Background(), u)
	if err == nil {
		t.Fatalf("expected ErrConflict, got nil")
	}
	if err != ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestCreate_DupOriginalRace_ReturnsExistingCode(t *testing.T) {
	store := newFakeStore()
	store.forceDupOrig = true
	store.existingOrig = "https://example.com/race"
	store.existingCode = "EXISTING00"

	svc := NewShortener(store, stubGen("NEWHERE000"))

	code, err := svc.Create(context.Background(), store.existingOrig)
	if err != nil {
		t.Fatalf("Create err: %v", err)
	}
	if code != store.existingCode {
		t.Fatalf("expected existing code %q, got %q", store.existingCode, code)
	}
}

func TestCreate_InvalidURL_ReturnsErrInvalidURL(t *testing.T) {
	store := newFakeStore()
	svc := NewShortener(store, stubGen("AAAAAAAAAA"))

	_, err := svc.Create(context.Background(), "ftp://bad.example") 
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != ErrInvalidURL {
		t.Fatalf("expected ErrInvalidURL, got %v", err)
	}
}

func TestResolve_Found(t *testing.T) {
	store := newFakeStore()
	store.byOrig["https://example.com/a"] = "AAAAAAAAAA"
	store.byCode["AAAAAAAAAA"] = "https://example.com/a"

	svc := NewShortener(store, stubGen("ignored"))
	u, err := svc.Resolve(context.Background(), "AAAAAAAAAA")
	if err != nil {
		t.Fatalf("Resolve err: %v", err)
	}
	if u != "https://example.com/a" {
		t.Fatalf("unexpected url: %q", u)
	}
}

func TestResolve_NotFound(t *testing.T) {
	store := newFakeStore()
	svc := NewShortener(store, stubGen("ignored"))

	_, err := svc.Resolve(context.Background(), "ZZZZZZZZZZ")
	if err == nil {
		t.Fatalf("expected ErrNotFound, got nil")
	}
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
