package app

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

func ExecutePeriodically(f func(), delay, interval time.Duration) {
	time.Sleep(delay)
	f()
	for range time.Tick(interval) {
		f()
	}
}

func InSlice[T comparable](x T, list []T) bool {
	for _, e := range list {
		if e == x {
			return true
		}
	}
	return false
}

func MergeMaps[T any](a map[string]T, bs ...map[string]T) map[string]T {
	m := make(map[string]T)
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

func SafeGetString(a map[string]any, key string) string {
	if a == nil {
		return ""
	}
	x, ok := a[key]
	if !ok {
		return ""
	}
	s, ok := x.(string)
	if !ok {
		return ""
	}
	return s
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

func Last[T any](s []T) T {
	return s[len(s)-1]
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

func TranslateError(msg string) string {
	for k, v := range globalErrorTranslations {
		if strings.Contains(msg, k) {
			return v
		}
	}

	return msg
}

func GenerateRandomID() string {
	salt := make([]byte, 32)
	rand.Read(salt)
	return hex.EncodeToString(salt)
}
