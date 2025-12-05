package geo

import (
	"net"

	"github.com/oschwald/geoip2-golang"
)

type Context struct {
	IP           string `json:"ip"`
	Country      string `json:"country"`
	Timezone     string `json:"timezone"`
	LangHeader   string `json:"langHeader"`
	TzCountry    string `json:"tzCountry"`
	LangCountry  string `json:"langCountry"`
	IsVpnSuspect bool   `json:"isVpnSuspect"`
	TrustLevel   string `json:"trustLevel"`
}

var geoDb *geoip2.Reader

// InitGeo loads the GeoLite2 database once at startup.
func InitGeo(dbPath string) error {
	db, err := geoip2.Open(dbPath)
	if err != nil {
		return err
	}
	geoDb = db
	return nil
}

func getCountryFromIP(ipStr string) string {
	if geoDb == nil {
		return ""
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ""
	}
	rec, err := geoDb.Country(ip)
	if err != nil {
		return ""
	}
	return rec.Country.IsoCode
}

// Very rough mapping; extend for your needs.
func getCountryFromTimezone(tz string) string {
	if tz == "" {
		return ""
	}
	switch tz {
	case "Asia/Karachi":
		return "PK"
	case "Asia/Kolkata":
		return "IN"
	case "Europe/Berlin":
		return "DE"
	}
	return ""
}

// Minimal parser: tries to extract "US" from strings like "en-US,en;q=0.9"
func getCountryFromAcceptLanguage(h string) string {
	for i := 0; i < len(h)-2; i++ {
		if h[i] == '-' && i+3 <= len(h) {
			return h[i+1 : i+3]
		}
	}
	return ""
}

func computeTrust(ipCountry, tzCountry, langCountry string, datacenter bool) string {
	if ipCountry == "" {
		return "unknown"
	}
	if datacenter {
		return "very_low"
	}
	sameTz := tzCountry != "" && tzCountry == ipCountry
	sameLang := langCountry != "" && langCountry == ipCountry

	if sameTz && sameLang {
		return "high"
	}
	if sameTz || sameLang {
		return "medium"
	}
	return "low"
}
