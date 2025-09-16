package core

import "context"

type CodeGenerator func(n int) (string, error)

type Shortener struct {
	store Store
	gen CodeGenerator
	tries int
}

func NewShortener(store Store, gen CodeGenerator) *Shortener {
	return &Shortener{store: store, gen: gen, tries: 6}
}


func (s *Shortener) Create(ctx context.Context, raw string) (string, error) {
    normalized, err := ValidateURL(raw)
    if err != nil {
        return "", ErrInvalidURL
    }

    if code, found, err := s.store.GetByOriginal(ctx, normalized); err != nil {
        return "", err
    } else if found {
        return code, nil
    }

    for i := 0; i < s.tries; i++ {
        code, err := s.gen(CodeLen)
        if err != nil {
            return "", err
        }
        if !IsValidCode(code) {
            continue
        }

        err = s.store.Create(ctx, code, normalized)
        switch err {
        case nil:
            return code, nil
        case ErrDupCode:
            continue
        case ErrDupOrigin:
            if c, found, e2 := s.store.GetByOriginal(ctx, normalized); e2 != nil {
                return "", e2
            } else if found {
                return c, nil
            }
            continue
        default:
            return "", err
        }
    }
    return "", ErrConflict
}

func (s *Shortener) Resolve(ctx context.Context, code string) (string, error) {
    if !IsValidCode(code) {
        return "", ErrNotFound
    }
    original, found, err := s.store.GetByCode(ctx, code)
    if err != nil {
        return "", err
    }
    if !found {
        return "", ErrNotFound
    }
    return original, nil
}
