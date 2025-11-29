package app

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

func InSlice[T comparable](x T, list []T) bool {
	for _, e := range list {
		if e == x {
			return true
		}
	}
	return false
}

func MergeMaps(a map[string]any, bs ...map[string]any) map[string]any {
	m := make(map[string]any)
	for k, v := range a {
		m[k] = v
	}
	for _, b := range bs {
		for k, v := range b {
			m[k] = v
		}
	}
	return m
}

func ExecutePeriodically(f func(), delay, interval time.Duration) {
	time.Sleep(delay)
	f()
	for range time.Tick(interval) {
		f()
	}
}

func Max[T any](s []T, greater func(T, T) bool) T {
	max := s[0]
	for _, x := range s {
		if greater(x, max) {
			max = x
		}
	}
	return max
}

func Map[T, R any](s []T, f func(T) R) (out []R) {
	for _, x := range s {
		out = append(out, f(x))
	}
	return out
}

func Filter[T any](s []T, f func(T) bool) (out []T) {
	for _, x := range s {
		if f(x) {
			out = append(out, x)
		}
	}
	return out
}

func translateError(msg string, t any) string {
	if v, ok := t.(ValidationTranslator); ok {
		for k, v := range v.ValidationTranslations() {
			if strings.Contains(msg, k) {
				return v
			}
		}
	}

	return msg
}

func generateRandomID() string {
	salt := make([]byte, 32)
	rand.Read(salt)
	return hex.EncodeToString(salt)
}
