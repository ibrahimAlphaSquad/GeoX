#!/usr/bin/env bash
set -euo pipefail

INITIAL_ARGC=$#

usage() {
  cat <<'USAGE'
Usage: download_geoip.sh [options]

Downloads GeoLite2 MMDB files into `geoip/` (default) or a specified dest.

By default (no flags) the script attempts to download standard MMDBs from
the GitHub repo `P3TERX/GeoLite.mmdb` (raw files from the `main` branch).

Options:
  -k, --key         MaxMind license key (or set MAXMIND_LICENSE_KEY env var)
  --products        Comma-separated product IDs (default: GeoLite2-City,GeoLite2-Country,GeoLite2-ASN)
  --dest            Destination directory (default: geoip)
  --github-url URL  Download a single file (raw .mmdb or tar.gz) from a GitHub URL
  --github-auto     Auto-download standard MMDBs from a GitHub repo (default repo: P3TERX/GeoLite.mmdb)
  --github-repo REPO Specify GitHub repo for --github-auto (format: owner/repo)
  -h, --help        Show this help and exit

Examples:
  # default (downloads from P3TERX/GeoLite.mmdb)
  ./scripts/download_geoip.sh

  # download a single raw mmdb from GitHub
  ./scripts/download_geoip.sh --github-url https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/main/GeoLite2-City.mmdb

  # use a MaxMind license key to download official archives
  MAXMIND_LICENSE_KEY=your_key ./scripts/download_geoip.sh
USAGE
}

KEY=""
DEST_DIR="geoip"
DEFAULT_PRODUCTS=("GeoLite2-City" "GeoLite2-Country" "GeoLite2-ASN")
PRODUCTS=()
GITHUB_URL=""
GITHUB_AUTO=0
GITHUB_REPO="P3TERX/GeoLite.mmdb"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -k|--key)
      KEY="$2"; shift 2;;
    --products)
      IFS=',' read -r -a PRODUCTS <<< "$2"; shift 2;;
    --dest)
      DEST_DIR="$2"; shift 2;;
    --github-url)
      GITHUB_URL="$2"; shift 2;;
    --github-auto)
      GITHUB_AUTO=1; shift 1;;
    --github-repo)
      GITHUB_REPO="$2"; shift 2;;
    -h|--help)
      usage; exit 0;;
    *)
      echo "Unknown argument: $1"; usage; exit 1;;
  esac
done

# If invoked with no args, default to GitHub auto-download
if [[ ${INITIAL_ARGC:-0} -eq 0 ]]; then
  GITHUB_AUTO=1
fi

if [[ -z "${GITHUB_URL:-}" && ${GITHUB_AUTO:-0} -eq 0 ]]; then
  if [[ -z "${KEY:-}" ]]; then
    KEY="${MAXMIND_LICENSE_KEY:-}"
  fi
  if [[ -z "${KEY:-}" ]]; then
    echo "Error: MaxMind license key not provided. Set MAXMIND_LICENSE_KEY or pass -k/--key, or use --github-url/--github-auto." >&2
    exit 2
  fi
fi

if [[ ${#PRODUCTS[@]} -eq 0 ]]; then
  PRODUCTS=("${DEFAULT_PRODUCTS[@]}")
fi

mkdir -p "$DEST_DIR"

TMPDIR=$(mktemp -d)
cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

download_file() {
  local url="$1" out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fSL --retry 3 --retry-delay 2 "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
  else
    echo "Error: neither curl nor wget is installed." >&2
    return 3
  fi
}

if [[ -n "${GITHUB_URL}" ]]; then
  echo "Downloading from GitHub URL: $GITHUB_URL -> $DEST_DIR"
  out="$TMPDIR/$(basename "$GITHUB_URL")"
  download_file "$GITHUB_URL" "$out"

  # If tarball, extract and move mmdb; if .mmdb, move directly
  if [[ "$out" == *.tar.gz ]] || file "$out" | grep -qi 'gzip compressed data'; then
    tar -xzf "$out" -C "$TMPDIR"
    mmdb=$(find "$TMPDIR" -maxdepth 3 -type f -name "*.mmdb" -print -quit || true)
    if [[ -z "${mmdb}" ]]; then
      echo "Warning: no .mmdb file found in the downloaded archive." >&2
      exit 4
    fi
    mv -f "$mmdb" "$DEST_DIR/$(basename "$mmdb")"
    echo "Saved: $DEST_DIR/$(basename "$mmdb")"
  else
    # Assume direct mmdb or other file
    if [[ $(basename "$out") == *.mmdb ]]; then
      mv -f "$out" "$DEST_DIR/$(basename "$out")"
      echo "Saved: $DEST_DIR/$(basename "$out")"
    else
      # try to detect archive type
      if file "$out" | grep -qi 'gzip compressed data'; then
        tar -xzf "$out" -C "$TMPDIR"
        mmdb=$(find "$TMPDIR" -maxdepth 3 -type f -name "*.mmdb" -print -quit || true)
        if [[ -n "${mmdb}" ]]; then
          mv -f "$mmdb" "$DEST_DIR/$(basename "$mmdb")"
          echo "Saved: $DEST_DIR/$(basename "$mmdb")"
        else
          echo "Warning: downloaded file is not an .mmdb and not an archive containing one." >&2
          exit 5
        fi
      else
        echo "Saving file as: $DEST_DIR/$(basename "$out")"
        mv -f "$out" "$DEST_DIR/$(basename "$out")"
      fi
    fi
  fi

  echo -e "\nAll done. The files are in: $DEST_DIR"
  exit 0
fi

if [[ ${GITHUB_AUTO:-0} -eq 1 ]]; then
  echo "Auto-downloading MMDBs from GitHub repo: ${GITHUB_REPO} -> $DEST_DIR"
  raw_base="https://raw.githubusercontent.com/${GITHUB_REPO}/main"

  for product in "${PRODUCTS[@]}"; do
    filename="${product}.mmdb"
    url_primary="${raw_base}/${filename}"
    url_fallback="https://github.com/${GITHUB_REPO}/raw/download/${filename}"
    out="$TMPDIR/${filename}"
    echo "-> Trying $url_primary"
    if download_file "$url_primary" "$out"; then
      mv -f "$out" "$DEST_DIR/$(basename "$out")"
      echo "Saved: $DEST_DIR/$(basename "$out")"
      continue
    fi

    echo "-> primary failed, trying fallback $url_fallback"
    if download_file "$url_fallback" "$out"; then
      mv -f "$out" "$DEST_DIR/$(basename "$out")"
      echo "Saved: $DEST_DIR/$(basename "$out")"
      continue
    fi

    echo "Warning: failed to download ${filename} from both primary and fallback URLs" >&2
  done

  echo -e "\nDone. Check $DEST_DIR for downloaded files."
  exit 0
fi

echo "Downloading products: ${PRODUCTS[*]} -> $DEST_DIR"

for product in "${PRODUCTS[@]}"; do
  echo -e "\n-> Downloading $product"
  url="https://download.maxmind.com/app/geoip_download?license_key=${KEY}&product_id=${product}&suffix=tar.gz"
  out="$TMPDIR/$(echo "$product" | sed 's/[^A-Za-z0-9._-]/_/g').tar.gz"

  download_file "$url" "$out"

  echo "Extracting..."
  tar -xzf "$out" -C "$TMPDIR"

  mmdb=$(find "$TMPDIR" -maxdepth 2 -type f -name "*.mmdb" -print -quit || true)
  if [[ -z "${mmdb}" ]]; then
    echo "Warning: no .mmdb file found for $product. Skipping." >&2
    continue
  fi

  dst="$DEST_DIR/$(basename "$mmdb")"
  mv -f "$mmdb" "$dst"
  echo "Saved: $dst"
done

echo -e "\nAll done. The files are in: $DEST_DIR"
# Remove stray Markdown code fences
#!/usr/bin/env bash
set -euo pipefail

INITIAL_ARGC=$#
usage() {
  cat <<'USAGE'
Usage: download_geoip.sh [-k KEY] [--products Product1,Product2] [--dest DIR]

Downloads MaxMind GeoLite2 databases (City, Country, ASN) into `geoip/`.

Options:
  -k, --key       MaxMind license key (or set MAXMIND_LICENSE_KEY env var)
  --products      Comma-separated product IDs (default: GeoLite2-City,GeoLite2-Country,GeoLite2-ASN)
  --dest          Destination directory (default: geoip)
  -h, --help      Show this help and exit

Example:
  #!/usr/bin/env bash
  set -euo pipefail

  usage() {
    cat <<'USAGE'
  Usage: download_geoip.sh [options]

  Downloads MaxMind GeoLite2 databases (City, Country, ASN) into `geoip/`,
  or downloads a raw/tarball MMDB from a provided GitHub URL, or auto-downloads
  standard MMDBs from a GitHub repo (default: P3TERX/GeoLite.mmdb).

  Options:
    -k, --key         MaxMind license key (or set MAXMIND_LICENSE_KEY env var)
    --products        Comma-separated product IDs (default: GeoLite2-City,GeoLite2-Country,GeoLite2-ASN)
    --dest            Destination directory (default: geoip)
    --github-url URL  Download a single file (raw .mmdb or tar.gz) from a GitHub URL instead of MaxMind
    --github-auto     Auto-download standard MMDBs from a GitHub repo (default repo: P3TERX/GeoLite.mmdb)
    --github-repo REPO Specify GitHub repo for --github-auto (format: owner/repo)
    -h, --help        Show this help and exit

  Examples:
    MAXMIND_LICENSE_KEY=abcd1234 ./scripts/download_geoip.sh
    ./scripts/download_geoip.sh --github-url https://raw.githubusercontent.com/P3TERX/GeoLite.mmdb/main/GeoLite2-City.mmdb
    ./scripts/download_geoip.sh --github-auto

  Notes:
    - Requires `curl` or `wget` and `tar` for archive extraction.
    - For MaxMind downloads you must have a valid license key.
  USAGE
  }

  KEY=""
  DEST_DIR="geoip"
  DEFAULT_PRODUCTS=("GeoLite2-City" "GeoLite2-Country" "GeoLite2-ASN")
  PRODUCTS=()
  GITHUB_URL=""
  GITHUB_AUTO=0
  GITHUB_REPO="P3TERX/GeoLite.mmdb"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -k|--key)
        KEY="$2"; shift 2;;
      --products)
        IFS=',' read -r -a PRODUCTS <<< "$2"; shift 2;;
      --dest)
        DEST_DIR="$2"; shift 2;;
      --github-url)
        GITHUB_URL="$2"; shift 2;;
      --github-auto)
        GITHUB_AUTO=1; shift 1;;
      --github-repo)
        GITHUB_REPO="$2"; shift 2;;
      -h|--help)
        usage; exit 0;;
      *)
        echo "Unknown argument: $1"; usage; exit 1;;
    esac
  done

  # If the script was invoked with no arguments, default to GitHub auto-download
  if [[ ${INITIAL_ARGC:-0} -eq 0 ]]; then
    GITHUB_AUTO=1
  fi

  if [[ -z "${GITHUB_URL:-}" && ${GITHUB_AUTO:-0} -eq 0 ]]; then
    if [[ -z "${KEY:-}" ]]; then
      KEY="${MAXMIND_LICENSE_KEY:-}"
    fi
    if [[ -z "${KEY:-}" ]]; then
      echo "Error: MaxMind license key not provided. Set MAXMIND_LICENSE_KEY or pass -k/--key, or use --github-url/--github-auto." >&2
      exit 2
    fi
  fi

  if [[ ${#PRODUCTS[@]} -eq 0 ]]; then
    PRODUCTS=("${DEFAULT_PRODUCTS[@]}")
  fi

  mkdir -p "$DEST_DIR"

  TMPDIR=$(mktemp -d)
  cleanup() { rm -rf "$TMPDIR"; }
  trap cleanup EXIT

  download_file() {
    local url="$1" out="$2"
    if command -v curl >/dev/null 2>&1; then
      curl -fSL --retry 3 --retry-delay 2 "$url" -o "$out"
    elif command -v wget >/dev/null 2>&1; then
      wget -qO "$out" "$url"
    else
      echo "Error: neither curl nor wget is installed." >&2
      return 3
    fi
  }

  if [[ -n "${GITHUB_URL}" ]]; then
    echo "Downloading from GitHub URL: $GITHUB_URL -> $DEST_DIR"
    out="$TMPDIR/$(basename "$GITHUB_URL")"
    download_file "$GITHUB_URL" "$out"

    # If tarball, extract and move mmdb; if .mmdb, move directly
    if [[ "$out" == *.tar.gz ]] || file "$out" | grep -qi 'gzip compressed data'; then
      tar -xzf "$out" -C "$TMPDIR"
      mmdb=$(find "$TMPDIR" -maxdepth 3 -type f -name "*.mmdb" -print -quit || true)
      if [[ -z "${mmdb}" ]]; then
        echo "Warning: no .mmdb file found in the downloaded archive." >&2
        exit 4
      fi
      mv -f "$mmdb" "$DEST_DIR/$(basename "$mmdb")"
      echo "Saved: $DEST_DIR/$(basename "$mmdb")"
    else
      # Assume direct mmdb
      if [[ $(basename "$out") != *.mmdb ]]; then
        # try to detect content type; if it's a gzip of mmdb, try extracting
        if file "$out" | grep -qi 'gzip compressed data'; then
          tar -xzf "$out" -C "$TMPDIR"
          mmdb=$(find "$TMPDIR" -maxdepth 3 -type f -name "*.mmdb" -print -quit || true)
          if [[ -n "${mmdb}" ]]; then
            mv -f "$mmdb" "$DEST_DIR/$(basename "$mmdb")"
            echo "Saved: $DEST_DIR/$(basename "$mmdb")"
          else
            echo "Warning: downloaded file is not an .mmdb and not an archive containing one." >&2
            exit 5
          fi
        else
          echo "Saving file as: $DEST_DIR/$(basename "$out")"
          mv -f "$out" "$DEST_DIR/$(basename "$out")"
        fi
      else
        mv -f "$out" "$DEST_DIR/$(basename "$out")"
        echo "Saved: $DEST_DIR/$(basename "$out")"
      fi
    fi

      echo -e "\nAll done. The files are in: $DEST_DIR"
    exit 0
  fi

  if [[ ${GITHUB_AUTO:-0} -eq 1 ]]; then
    echo "Auto-downloading MMDBs from GitHub repo: ${GITHUB_REPO} -> $DEST_DIR"
    raw_base="https://raw.githubusercontent.com/${GITHUB_REPO}/main"

    for product in "${PRODUCTS[@]}"; do
      filename="${product}.mmdb"
      url="${raw_base}/${filename}"
      out="$TMPDIR/${filename}"
      echo "-> Trying $url"
      if download_file "$url" "$out"; then
        mv -f "$out" "$DEST_DIR/$(basename "$out")"
        echo "Saved: $DEST_DIR/$(basename "$out")"
      else
        echo "Warning: failed to download $url" >&2
      fi
    done

    echo "\nDone. Check $DEST_DIR for downloaded files."
    exit 0
  fi

  echo "Downloading products: ${PRODUCTS[*]} -> $DEST_DIR"

  for product in "${PRODUCTS[@]}"; do
    echo "\n-> Downloading $product"
    url="https://download.maxmind.com/app/geoip_download?license_key=${KEY}&product_id=${product}&suffix=tar.gz"
    out="$TMPDIR/$(echo "$product" | sed 's/[^A-Za-z0-9._-]/_/g').tar.gz"

    download_file "$url" "$out"

    echo "Extracting..."
    tar -xzf "$out" -C "$TMPDIR"

    mmdb=$(find "$TMPDIR" -maxdepth 2 -type f -name "*.mmdb" -print -quit || true)
    if [[ -z "${mmdb}" ]]; then
      echo "Warning: no .mmdb file found for $product. Skipping." >&2
      continue
    fi

    dst="$DEST_DIR/$(basename "$mmdb")"
    mv -f "$mmdb" "$dst"
    echo "Saved: $dst"
  done

  echo -e "\nAll done. The files are in: $DEST_DIR"
