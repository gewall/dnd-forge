package httpapi

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/time/rate"

	"github.com/gewall/dnd-forge/api/internal/identity"
)

// limit is a token bucket: Every is the refill interval of one token.
type limit struct {
	Every time.Duration
	Burst int
}

// policies are the named per-operation limits. In-memory buckets are enough
// because the API runs as a single instance.
var policies = map[string]limit{
	// login, register, refresh: ±10 percobaan per menit per IP.
	"auth": {Every: 6 * time.Second, Burst: 10},
	// generate portrait: di samping kuota harian, maks 1 request per 10 detik.
	"generate": {Every: 10 * time.Second, Burst: 2},
}

// globalPerIP protects every route from a single noisy client.
var globalPerIP = limit{Every: 50 * time.Millisecond, Burst: 60}

const limiterIdleTTL = 15 * time.Minute

type bucket struct {
	lim      *rate.Limiter
	lastSeen time.Time
}

// limiters holds one bucket per (policy, key) and forgets idle ones.
type limiters struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

func newLimiters() *limiters {
	return &limiters{buckets: map[string]*bucket{}, lastSweep: nowFunc()}
}

// allow reports whether the caller identified by key may proceed under l,
// and how long to wait otherwise.
func (ls *limiters) allow(key string, l limit) (bool, time.Duration) {
	now := nowFunc()
	ls.mu.Lock()
	defer ls.mu.Unlock()

	if now.Sub(ls.lastSweep) > limiterIdleTTL {
		for k, b := range ls.buckets {
			if now.Sub(b.lastSeen) > limiterIdleTTL {
				delete(ls.buckets, k)
			}
		}
		ls.lastSweep = now
	}

	b, ok := ls.buckets[key]
	if !ok {
		b = &bucket{lim: rate.NewLimiter(rate.Every(l.Every), l.Burst)}
		ls.buckets[key] = b
	}
	b.lastSeen = now
	r := b.lim.ReserveN(now, 1)
	if delay := r.DelayFrom(now); delay > 0 {
		r.CancelAt(now)
		return false, delay
	}
	return true, 0
}

func (s *Server) limitByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, wait := s.limits.allow("ip:"+ClientIP(r.Context()), globalPerIP); !ok {
			w.Header().Set("Retry-After", retryAfter(wait))
			writeProblem(w, http.StatusTooManyRequests, "terlalu banyak request, coba lagi sebentar")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// rateLimit applies the policy an operation declared with RateLimited. It
// runs after authenticate, so signed-in users are limited per account.
func (s *Server) rateLimit(ctx huma.Context, next func(huma.Context)) {
	name, _ := ctx.Operation().Metadata[metaRateLimit].(string)
	if name == "" {
		next(ctx)
		return
	}
	key := name + ":ip:" + ClientIP(ctx.Context())
	if p, ok := identity.FromContext(ctx.Context()); ok {
		key = name + ":user:" + p.UserID.String()
	}
	if ok, wait := s.limits.allow(key, policies[name]); !ok {
		ctx.SetHeader("Retry-After", retryAfter(wait))
		s.writeErr(ctx, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi sebentar")
		return
	}
	next(ctx)
}

func retryAfter(d time.Duration) string {
	return strconv.Itoa(int(d.Round(time.Second)/time.Second) + 1)
}
