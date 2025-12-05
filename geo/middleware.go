package geo

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type ctxKey int

const geoKey ctxKey = iota

// WithContext stores the Geo context into the request context.
func withContext(ctx context.Context, geo *Context) context.Context {
	return context.WithValue(ctx, geoKey, geo)
}

// FromRequest retrieves the Geo context from the request.
func FromRequest(r *http.Request) *Context {
	if v := r.Context().Value(geoKey); v != nil {
		if g, ok := v.(*Context); ok {
			return g
		}
	}
	return nil
}

// Middleware computes geo info and adds it to the request context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract client IP (respect X-Forwarded-For if behind proxy)
		xff := r.Header.Get("X-Forwarded-For")
		ip := ""
		if xff != "" {
			parts := strings.Split(xff, ",")
			ip = strings.TrimSpace(parts[0])
		} else {
			// r.RemoteAddr usually "ip:port"
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			ip = host
		}

		// 2. Signal sources
		countryFromIp := getCountryFromIP(ip)
		tzHeader := r.Header.Get("X-Timezone")       // set by frontend
		langHeader := r.Header.Get("Accept-Language") // from browser

		tzCountry := getCountryFromTimezone(tzHeader)
		langCountry := getCountryFromAcceptLanguage(langHeader)

		dc := isDatacenterIP(ip)
		trust := computeTrust(countryFromIp, tzCountry, langCountry, dc)

		vpnSuspect := dc ||
			(countryFromIp != "" && tzCountry != "" && countryFromIp != tzCountry) ||
			(countryFromIp != "" && langCountry != "" && countryFromIp != langCountry)

		geo := &Context{
			IP:           ip,
			Country:      countryFromIp,
			Timezone:     tzHeader,
			LangHeader:   langHeader,
			TzCountry:    tzCountry,
			LangCountry:  langCountry,
			IsVpnSuspect: vpnSuspect,
			TrustLevel:   trust,
		}

		ctx := withContext(r.Context(), geo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
