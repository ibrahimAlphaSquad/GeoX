# GeoIP Country Detection

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

This guide explains how to implement IP-based country detection **using only Go** and a **local GeoIP database**. No third‑party services or client‑side logic are required.

It includes:

* 📦 Project structure
* 🏗️ Backend implementation
* 🔍 VPN / Datacenter suspicion logic
* 🧪 Testing instructions

---

## 1. Overview

Your Go backend can detect a user’s country by:

* Extracting their IP address
* Looking it up in a local **GeoLite2 Country** database (MMDB)
* Applying VPN / proxy detection heuristics
* Computing a trust level based on available signals

This solution makes **no external network calls**.

---

## 2. Project Structure

```
GeoX/
  go.mod
  main.go
  geo/
    geo.go
    datacenter.go
    middleware.go
  geoip/
    GeoLite2-Country.mmdb
```

Download **GeoLite2-Country.mmdb** from MaxMind and place it in `geoip/`.

---

## 3. Backend Implementation (Go)

### `geo.go`

Implements:

* Loading the GeoIP database
* Converting IP → country
* Parsing language headers
* Mapping timezone identifiers → country codes
* Calculating a risk/trust level

### `datacenter.go`

Provides a list of datacenter CIDRs (AWS, GCP, Cloudflare, etc.) to detect suspicious/VPN-like IP ranges.

### `middleware.go`

Adds a `geo.Context` object to each HTTP request containing:

* IP address
* Country
* Timezone header (optional)
* Language header
* Timezone-derived country
* Language-derived country
* VPN suspicion flag
* Trust score

### `main.go`

Initializes all components and exposes a test endpoint (`/info`).

---

## 4. Example Response

Calling:

```
curl http://localhost:8080/info
```

returns:

```json
{
  "ip": "203.0.113.10",
  "country": "DE",
  "timezone": "Europe/Berlin",
  "langHeader": "de-DE,de;q=0.9",
  "tzCountry": "DE",
  "langCountry": "DE",
  "isVpnSuspect": false,
  "trustLevel": "high"
}
```

---

## 5. Testing Scenarios

### Fake the client IP

```
curl -H "X-Forwarded-For: 203.0.113.10" http://localhost:8080/info
```

### Add timezone + language

```
curl \
  -H "X-Forwarded-For: 203.0.113.10" \
  -H "X-Timezone: Europe/Berlin" \
  -H "Accept-Language: de-DE" \
  http://localhost:8080/info
```

### Suspicious (mismatched timezone)

```
curl \
  -H "X-Forwarded-For: 203.0.113.10" \
  -H "X-Timezone: Asia/Karachi" \
  http://localhost:8080/info
```

---

## 6. How It Works

### IP addresses themselves do not store country information.

GeoIP detection works because:

* IANA allocates IP blocks to RIRs
* RIRs allocate IP ranges to ISPs
* ISPs operate in known countries

GeoIP providers aggregate:

* WHOIS records
* BGP routing tables
* ISP metadata
* Community corrections

The result is stored in an optimized `.mmdb` file that Go can load very efficiently.

---

## 7. Extending the System

You can extend this design by adding:

* City-level detection (with GeoLite2 City)
* Better VPN / proxy heuristics
* Per-country access control
* Dynamic rate limiting based on country
* Risk scoring using multiple signals

---

## License

This project is licensed under the MIT License — see the `LICENSE` file for details.

## Download GeoIP databases

A helper script is provided to fetch GeoLite2 MMDB files. By default (no flags) it will try to download standard MMDBs from the `P3TERX/GeoLite.mmdb` GitHub repository and place them in the `geoip/` directory.

Usage:

```bash
chmod +x ./scripts/download_geoip.sh
./scripts/download_geoip.sh
```

You can also download a single raw MMDB directly from a GitHub URL:

```bash
./scripts/download_geoip.sh --github-url https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/main/GeoLite2-City.mmdb
```

Or use a MaxMind license key to download official GeoLite2 archives:

```bash
MAXMIND_LICENSE_KEY=your_key_here ./scripts/download_geoip.sh
```

The script requires `curl` or `wget` and `tar` for archive extraction.

## API Documentation (Swagger)

The project includes an OpenAPI (Swagger) spec and a small Swagger UI.

Start the server and visit `http://localhost:8082/docs` to view interactive API documentation.

Files added:

- `openapi.yaml` — OpenAPI 3.0 spec for the basic endpoints.
- `docs/swagger.html` — Swagger UI wrapper that loads the spec from `/openapi.yaml`.