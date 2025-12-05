package geo

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type ctxKey int

const geoKey ctxKey = iota

// Middleware attaches a *geo.Info to each request.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ipStr := extractIP(r)
		info := lookupAll(ipStr)

		// Headers (timezone, language, etc.)
		enrichFromHeaders(info, r)

		// User-Agent, device, browser
		enrichFromUserAgent(info, r)

		// Datacenter / VPN / trust
		parsed := net.ParseIP(ipStr)
		info.IsDatacenterIP = isDatacenterIP(parsed)
		info.IsVPN = looksLikeVPN(info)
		info.TrustLevel = computeTrust(info)

		ctx := context.WithValue(r.Context(), geoKey, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractIP considers X-Forwarded-For then falls back to RemoteAddr.
func extractIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// FromRequest returns *Info attached by the middleware.
func FromRequest(r *http.Request) *Info {
	if v := r.Context().Value(geoKey); v != nil {
		if info, ok := v.(*Info); ok {
			return info
		}
	}
	return nil
}
