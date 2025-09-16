package core

import (
	"crypto/rand"
	"strings"
)

const Alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
const CodeLen = 10
func NewCode(n int) (string, error) {
    buf := make([]byte, n)
    for i := 0; i < n; i++ {
        var b [1]byte
        for {
            if _, err := rand.Read(b[:]); err != nil {
                return "", err
            }
            if int(b[0]) < 252 { 
                buf[i] = Alphabet[int(b[0])%len(Alphabet)]
                break
            }
        }
    }
    return string(buf), nil
}

func IsValidCode(s string) bool {
	if len(s) != CodeLen {
		return false
	}
	for _, r := range s {
        if !strings.ContainsRune(Alphabet, r) {
            return false
        }
    }
    return true
}