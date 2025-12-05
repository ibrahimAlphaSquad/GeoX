package geo

import (
    "net"
    "net/http"
    "strings"

    "github.com/oschwald/geoip2-golang"
)

type Info struct {
    // IP
    IP string `json:"ip"`

    // Country DB fields
    Country            string `json:"country"`
    RegisteredCountry  string `json:"registeredCountry"`
    RepresentedCountry string `json:"representedCountry"`

    // City DB fields
    City           string  `json:"city"`
    State          string  `json:"state"`
    PostalCode     string  `json:"postalCode"`
    Latitude       float64 `json:"lat"`
    Longitude      float64 `json:"lon"`
    AccuracyRadius uint16  `json:"accuracyRadius"`
    Continent      string  `json:"continent"`

    // ASN DB fields
    ASN     uint   `json:"asn"`
    ASNOrg  string `json:"asnOrg"`
    Network string `json:"network"`

    // Headers
    AcceptLanguage string `json:"acceptLanguage"`
    TimezoneHeader string `json:"timezoneHeader"`

    // Derived
    TZCountry   string `json:"tzCountry"`
    LangCountry string `json:"langCountry"`

    // Security / risk
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

    // Country DB
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
        }
    }

    // City DB
    if cityDB != nil {
        if rec, err := cityDB.City(ip); err == nil {
            if name, ok := rec.City.Names["en"]; ok {
                info.City = name
            }
            if len(rec.Subdivisions) > 0 {
                if name, ok := rec.Subdivisions[0].Names["en"]; ok {
                    info.State = name
                }
            }
            info.PostalCode = rec.Postal.Code
            info.Latitude = rec.Location.Latitude
            info.Longitude = rec.Location.Longitude
            info.AccuracyRadius = rec.Location.AccuracyRadius
            if info.Continent == "" && rec.Continent.Code != "" {
                info.Continent = rec.Continent.Code
            }
        }
    }

    // ASN DB
    if asnDB != nil {
        if rec, err := asnDB.ASN(ip); err == nil {
            info.ASN = rec.AutonomousSystemNumber
            info.ASNOrg = rec.AutonomousSystemOrganization
            // Some versions of the geoip2-golang ASN type do not include a Network field.
            // If your version provides network information (e.g. as a different field), update
            // this code to set info.Network accordingly. Otherwise leave it empty.
        }
    }

    return info
}

// --- Derived data from headers ---

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
    info.TZCountry = deriveTZCountry(info.TimezoneHeader)
    info.LangCountry = deriveLangCountry(info.AcceptLanguage)
}
