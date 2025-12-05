package geo

import (
    "net"
    "strings"
)

var datacenterCidrs []*net.IPNet

// Common hosting / VPN / datacenter ASNs or brands.
var vpnASNKeywords = []string{
    "VULTR",
    "DIGITALOCEAN",
    "HETZNER",
    "OVH",
    "CONTABO",
    "GODADDY",
    "HOSTWINDS",
    "LINODE",
    "LEASEWEB",
    "AMAZON",
    "AMAZON.COM",
    "AMAZON TECHNOLOGIES",
    "GOOGLE",
    "MICROSOFT",
    "CLOUDFLARE",
}

// InitDatacenter sets a few broad datacenter ranges (you can extend).
func InitDatacenter() {
    cidrs := []string{
        "34.0.0.0/8",    // sample: Google-ish
        "52.0.0.0/8",    // sample: AWS-ish
        "104.16.0.0/12", // sample: Cloudflare-ish
    }
    for _, c := range cidrs {
        if _, network, err := net.ParseCIDR(c); err == nil {
            datacenterCidrs = append(datacenterCidrs, network)
        }
    }
}

func isDatacenterIP(ip net.IP) bool {
    if ip == nil {
        return false
    }
    for _, n := range datacenterCidrs {
        if n.Contains(ip) {
            return true
        }
    }
    return false
}

// looksLikeVPN uses ASN + mismatches + accuracy radius.
func looksLikeVPN(info *Info) bool {
    if info == nil {
        return false
    }

    // ASN org check
    upperOrg := strings.ToUpper(info.ASNOrg)
    for _, kw := range vpnASNKeywords {
        if strings.Contains(upperOrg, kw) {
            return true
        }
    }

    // Country vs Timezone / Language mismatch
    if info.Country != "" {
        if info.TZCountry != "" && info.TZCountry != info.Country {
            return true
        }
        if info.LangCountry != "" && info.LangCountry != info.Country {
            return true
        }
    }

    // Very large accuracy radius (very rough heuristic)
    if info.AccuracyRadius > 500 {
        return true
    }

    return false
}
