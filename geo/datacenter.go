package geo

import "net"

var datacenterCidrs []*net.IPNet

// Call this once at startup.
func InitDatacenterCidrs() {
	// Example ranges: you should tune this for your needs.
	cidrs := []string{
		"34.0.0.0/8",    // sample: Google Cloud-ish
		"52.0.0.0/8",    // sample: AWS-ish
		"104.16.0.0/12", // sample: Cloudflare-ish
	}
	for _, c := range cidrs {
		_, network, err := net.ParseCIDR(c)
		if err == nil {
			datacenterCidrs = append(datacenterCidrs, network)
		}
	}
}

func isDatacenterIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
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
