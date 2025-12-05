package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"

	"GeoX/geo"
)

func main() {
    // Init GeoIP databases
    if err := geo.Init(
        "./geoip/GeoLite2-Country.mmdb",
        "./geoip/GeoLite2-City.mmdb",
        "./geoip/GeoLite2-ASN.mmdb",
    ); err != nil {
        log.Fatalf("failed to init geo dbs: %v", err)
    }

    // Init datacenter CIDR heuristics
    geo.InitDatacenter()

    mux := http.NewServeMux()

    // Debug endpoint → full dump of all geo info
    mux.HandleFunc("/debug/geo", func(w http.ResponseWriter, r *http.Request) {
        info := geo.FromRequest(r)
        w.Header().Set("Content-Type", "application/json")
        if info == nil {
            w.WriteHeader(http.StatusInternalServerError)
            _ = json.NewEncoder(w).Encode(map[string]string{"error": "geo info missing"})
            return
        }
        _ = json.NewEncoder(w).Encode(info)
    })

    // Example normal endpoint
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        info := geo.FromRequest(r)
        if info != nil {
            w.Header().Set("X-Country", info.Country)
            vpn := "false"
            if info.IsVPN {
                vpn = "true"
            }
            w.Header().Set("X-VPN-Suspect", vpn)
        }
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("Hello from GeoX!"))
    })

    addr := ":8082"
    if v := os.Getenv("PORT"); v != "" {
        addr = ":" + v
    }

    log.Printf("Listening on %s", addr)
    if err := http.ListenAndServe(addr, geo.Middleware(mux)); err != nil {
        log.Fatal(err)
    }
}
