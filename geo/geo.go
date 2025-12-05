package geo

import (
	"net"
	"net/http"
	"strings"

	uaLib "github.com/mssola/user_agent"
	"github.com/oschwald/geoip2-golang"
)

type Info struct {
	// === Core IP ===
	IP string `json:"ip"`

	// === Country / Geo ===
	Country            string `json:"country"`
	RegisteredCountry  string `json:"registeredCountry"`
	RepresentedCountry string `json:"representedCountry"`

	City           string  `json:"city"`
	State          string  `json:"state"`
	StateCode      string  `json:"stateCode"`
	PostalCode     string  `json:"postalCode"`
	Latitude       float64 `json:"lat"`
	Longitude      float64 `json:"lon"`
	AccuracyRadius uint16  `json:"accuracyRadius"`
	Continent      string  `json:"continent"`
	Timezone       string  `json:"timezone"`
	MetroCode      uint    `json:"metroCode"`

	// MaxMind traits
	IsAnonymousProxy    bool   `json:"isAnonymousProxy"`
	IsSatelliteProvider bool   `json:"isSatelliteProvider"`
	Organization        string `json:"organization"`
	Domain              string `json:"domain"`
	ConnectionType      string `json:"connectionType"`

	// === ASN / Network ===
	ASN     uint   `json:"asn"`
	ASNOrg  string `json:"asnOrg"`
	Network string `json:"network"`

	// === Request headers (raw) ===
	AcceptLanguage string `json:"acceptLanguage"`
	TimezoneHeader string `json:"timezoneHeader"`

	UserAgent      string `json:"userAgent"`
	Accept         string `json:"accept"`
	AcceptEncoding string `json:"acceptEncoding"`
	AcceptCharset  string `json:"acceptCharset"`
	DNT            string `json:"dnt"`

	SecCHUA         string `json:"secChUa"`
	SecCHUAMobile   string `json:"secChUaMobile"`
	SecCHUAPlatform string `json:"secChUaPlatform"`
	XRequestedWith  string `json:"xRequestedWith"`
	Referer         string `json:"referer"`
	Origin          string `json:"origin"`

	// === Derived from headers ===
	TZCountry   string `json:"tzCountry"`
	LangCountry string `json:"langCountry"`

	// === Device / Browser ===
	DeviceType     string `json:"deviceType"` // "mobile","tablet","desktop","bot","unknown"
	OS             string `json:"os"`
	Browser        string `json:"browser"`
	BrowserVersion string `json:"browserVersion"`
	IsMobile       bool   `json:"isMobile"`
	IsBot          bool   `json:"isBot"`
	IsHeadless     bool   `json:"isHeadless"`
	IsAutomation   bool   `json:"isAutomation"`

	// === Risk / Security ===
	IsDatacenterIP bool   `json:"isDatacenter"`
	IsVPN          bool   `json:"isVPN"`
	TrustLevel     string `json:"trustLevel"`
}

var (
	countryDB *geoip2.Reader
	cityDB    *geoip2.Reader
	asnDB     *geoip2.Reader
)

// Init loads all MMDB files.
func Init(countryPath, cityPath, asnPath string) error {
	var err error
	countryDB, err = geoip2.Open(countryPath)
	if err != nil {
		return err
	}
	cityDB, err = geoip2.Open(cityPath)
	if err != nil {
		return err
	}
	asnDB, err = geoip2.Open(asnPath)
	if err != nil {
		return err
	}
	return nil
}

// lookupAll gets Country + City + ASN info for an IP.
func lookupAll(ipStr string) *Info {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return &Info{IP: ipStr}
	}

	info := &Info{IP: ipStr}

	// --- Country DB ---
	if countryDB != nil {
		if rec, err := countryDB.Country(ip); err == nil {
			if rec.Country.IsoCode != "" {
				info.Country = rec.Country.IsoCode
			}
			if rec.RegisteredCountry.IsoCode != "" {
				info.RegisteredCountry = rec.RegisteredCountry.IsoCode
			}
			if rec.RepresentedCountry.IsoCode != "" {
				info.RepresentedCountry = rec.RepresentedCountry.IsoCode
			}
			if rec.Continent.Code != "" {
				info.Continent = rec.Continent.Code
			}
			// Traits may be partially available even in GeoLite
			info.IsAnonymousProxy = rec.Traits.IsAnonymousProxy
			info.IsSatelliteProvider = rec.Traits.IsSatelliteProvider
		}
	}

	// --- City DB ---
	if cityDB != nil {
		if rec, err := cityDB.City(ip); err == nil {
			if name, ok := rec.City.Names["en"]; ok {
				info.City = name
			}
			if len(rec.Subdivisions) > 0 {
				if name, ok := rec.Subdivisions[0].Names["en"]; ok {
					info.State = name
				}
				info.StateCode = rec.Subdivisions[0].IsoCode
			}
			info.PostalCode = rec.Postal.Code
			info.Latitude = rec.Location.Latitude
			info.Longitude = rec.Location.Longitude
			info.AccuracyRadius = rec.Location.AccuracyRadius
			info.Timezone = rec.Location.TimeZone
			info.MetroCode = rec.Location.MetroCode

			if info.Continent == "" && rec.Continent.Code != "" {
				info.Continent = rec.Continent.Code
			}

			// Traits also exist here, override if present
			info.IsAnonymousProxy = info.IsAnonymousProxy || rec.Traits.IsAnonymousProxy
			info.IsSatelliteProvider = info.IsSatelliteProvider || rec.Traits.IsSatelliteProvider
		}
	}
	// if asnDB != nil {
	// 	if rec, err := asnDB.ASN(ip); err == nil {
	// 		info.ASN = rec.AutonomousSystemNumber
	// 		info.ASNOrg = rec.AutonomousSystemOrganization
	// 		if rec.Network != nil {
	// 			info.Network = rec.Network.String()
	// 		}
	// 	}
	// }

	if asnDB != nil {
	    if rec, err := asnDB.ASN(ip); err == nil {
	        info.ASN = rec.AutonomousSystemNumber
	        info.ASNOrg = rec.AutonomousSystemOrganization
	        // github.com/oschwald/geoip2-golang's ASN record does not include a Network field,
	        // so leave Info.Network empty here (or derive it separately if you have that data).
	        info.Network = ""
	    }
	}

	return info
}

// Derived data from headers ---
func deriveTZCountry(tz string) string {
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
	case "Europe/London":
		return "GB"
	case "America/New_York":
		return "US"
	case "America/Los_Angeles":
		return "US"
	}
	return ""
}

func deriveLangCountry(h string) string {
	if h == "" {
		return ""
	}
	parts := strings.Split(h, ",")
	primary := strings.TrimSpace(parts[0]) // e.g. "en-US"
	idx := strings.Index(primary, "-")
	if idx == -1 || idx+1 >= len(primary) {
		return ""
	}
	return primary[idx+1:]
}

// computeTrust gives a simple risk/trust score.
func computeTrust(info *Info) string {
	if info.Country == "" {
		return "unknown"
	}
	if info.IsDatacenterIP || info.IsVPN {
		return "very_low"
	}

	sameTz := info.TZCountry != "" && info.TZCountry == info.Country
	sameLang := info.LangCountry != "" && info.LangCountry == info.Country

	if sameTz && sameLang {
		return "high"
	}
	if sameTz || sameLang {
		return "medium"
	}
	return "low"
}

// enrichFromHeaders fills header-based fields.
func enrichFromHeaders(info *Info, r *http.Request) {
	info.AcceptLanguage = r.Header.Get("Accept-Language")
	info.TimezoneHeader = r.Header.Get("X-Timezone")

	info.Accept = r.Header.Get("Accept")
	info.AcceptEncoding = r.Header.Get("Accept-Encoding")
	info.AcceptCharset = r.Header.Get("Accept-Charset")
	info.DNT = r.Header.Get("DNT")

	info.SecCHUA = r.Header.Get("Sec-CH-UA")
	info.SecCHUAMobile = r.Header.Get("Sec-CH-UA-Mobile")
	info.SecCHUAPlatform = r.Header.Get("Sec-CH-UA-Platform")
	info.XRequestedWith = r.Header.Get("X-Requested-With")
	info.Referer = r.Header.Get("Referer")
	info.Origin = r.Header.Get("Origin")

	info.TZCountry = deriveTZCountry(info.TimezoneHeader)
	info.LangCountry = deriveLangCountry(info.AcceptLanguage)
}

func enrichFromUserAgent(info *Info, r *http.Request) {
	uaStr := r.Header.Get("User-Agent")
	info.UserAgent = uaStr

	if uaStr == "" {
		info.DeviceType = "unknown"
		return
	}

	ua := uaLib.New(uaStr)

	info.OS = ua.OS()
	name, version := ua.Browser()
	info.Browser = name
	info.BrowserVersion = version
	info.IsMobile = ua.Mobile()
	info.IsBot = ua.Bot()

	// Device type heuristic
	if info.IsBot {
		info.DeviceType = "bot"
	} else if info.IsMobile {
		info.DeviceType = "mobile"
	} else if strings.Contains(strings.ToLower(info.SecCHUA), "tablet") {
		info.DeviceType = "tablet"
	} else {
		info.DeviceType = "desktop"
	}

	lowerUA := strings.ToLower(uaStr)

	// Headless / automation hints
	if strings.Contains(lowerUA, "headless") ||
		strings.Contains(lowerUA, "puppeteer") ||
		strings.Contains(lowerUA, "playwright") {
		info.IsHeadless = true
		info.IsAutomation = true
	}

	if strings.Contains(lowerUA, "selenium") ||
		strings.Contains(lowerUA, "webdriver") {
		info.IsAutomation = true
	}
}
