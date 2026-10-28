package httpapi

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

// HeaderClientIP is set by Next.js to the visitor's IP address.
const HeaderClientIP = "X-Client-IP"

var nowFunc = time.Now

type client struct {
	ip      string
	trusted bool // request carried a valid internal key
}

type clientKey struct{}

func withClient(ctx context.Context, c client) context.Context {
	return context.WithValue(ctx, clientKey{}, c)
}

func clientFrom(ctx context.Context) client {
	c, _ := ctx.Value(clientKey{}).(client)
	return c
}

// ClientIP returns the visitor IP recorded for the request in ctx.
func ClientIP(ctx context.Context) string { return clientFrom(ctx).ip }

// clientIP picks the visitor address. The API only listens on 127.0.0.1
// behind Nginx, so X-Real-IP (set by Nginx) is trustworthy; X-Client-IP is
// honoured only for requests proven to come from Next.js.
func clientIP(r *http.Request, trusted bool) string {
	if trusted {
		if ip, ok := parseIP(r.Header.Get(HeaderClientIP)); ok {
			return ip
		}
	}
	if ip, ok := parseIP(r.Header.Get("X-Real-IP")); ok {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func parseIP(s string) (string, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(s))
	if err != nil {
		return "", false
	}
	return addr.Unmap().String(), true
}
