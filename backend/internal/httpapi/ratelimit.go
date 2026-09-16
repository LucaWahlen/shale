package httpapi

import (
	"net/http"
	"sync"
	"time"

	"shale/internal/domain"
)

const defaultWindow = time.Minute

type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
	now      func() time.Time
}

func newLoginLimiter(limit int, window time.Duration) *loginLimiter {
	return &loginLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		now:      time.Now,
	}
}

func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)

	timestamps := l.attempts[key]
	kept := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= l.limit {
		l.attempts[key] = kept
		if len(l.attempts) > 4096 {
			l.purgeLocked(cutoff)
		}
		return false
	}
	l.attempts[key] = append(kept, now)
	return true
}

func (l *loginLimiter) purgeLocked(cutoff time.Time) {
	for key, timestamps := range l.attempts {
		kept := timestamps[:0]
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				kept = append(kept, ts)
			}
		}
		if len(kept) == 0 {
			delete(l.attempts, key)
		} else {
			l.attempts[key] = kept
		}
	}
}

func (l *loginLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			writeError(w, domain.NewError(domain.KindRateLimited, "too many login attempts, try again later"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
