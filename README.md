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