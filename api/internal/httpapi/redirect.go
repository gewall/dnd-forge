package httpapi

import (
	"strconv"
	"time"
)

// Redirect is the output of operations that send the client to a presigned
// object URL.
type Redirect struct {
	Location     string `header:"Location"`
	CacheControl string `header:"Cache-Control"`
}

// NewRedirect points to url, which stays valid for ttl, and lets browsers
// reuse the redirect for most of that lifetime.
func NewRedirect(url string, ttl time.Duration) *Redirect {
	maxAge := int((ttl - 10*time.Minute).Seconds())
	return &Redirect{Location: url, CacheControl: "public, max-age=" + strconv.Itoa(max(maxAge, 0))}
}
