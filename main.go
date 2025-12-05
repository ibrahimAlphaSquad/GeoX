package main

import (
	"encoding/json"
	"log"
	"net/http"

	"GeoX/geo"
)

func main() {
	// 1. Init GeoIP database
	if err := geo.InitGeo("./geoip/GeoLite2-Country.mmdb"); err != nil {
		log.Fatalf("geo init error: %v", err)
	}

	// 2. Init datacenter CIDR list
	geo.InitDatacenterCidrs()

	mux := http.NewServeMux()

	// Simple test endpoint
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		g := geo.FromRequest(r)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(g)
	})

	log.Println("Listening on :8082")
	if err := http.ListenAndServe(":8082", geo.Middleware(mux)); err != nil {
		log.Fatal(err)
	}
}
