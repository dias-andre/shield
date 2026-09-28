#!/bin/sh

set -eu

DEFAULT_REPO="dias-andre/shield"
DEFAULT_VERSION="0.2.0"
REPO=${SHIELD_REPO:-$DEFAULT_REPO}
VERSION=${SHIELD_VERSION:-$DEFAULT_VERSION}
BIN_DIR=${SHIELD_BIN_DIR:-"${HOME:?HOME must be set}/.local/bin"}
SYSTEMD_DIR=${SHIELD_SYSTEMD_DIR:-"${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"}
START_SERVICE=${SHIELD_START_SERVICE:-1}

usage() {
    cat <<'EOF'
Install Shield in the current user's home directory.

Usage: install.sh [options]

Options:
  --repo OWNER/REPOSITORY  GitHub repository (default: dias-andre/shield)
  --version VERSION        Release tag (default: 0.2.0)
  --bin-dir DIR            Binary directory (default: ~/.local/bin)
  --systemd-dir DIR        User unit directory (default: ~/.config/systemd/user)
  --no-start               Install the unit without enabling or starting it
  -h, --help               Show this help

Environment equivalents: SHIELD_REPO, SHIELD_VERSION, SHIELD_BIN_DIR,
SHIELD_SYSTEMD_DIR, SHIELD_START_SERVICE (0 or 1).
EOF
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --repo) REPO=${2:?--repo requires a value}; shift 2 ;;
        --version) VERSION=${2:?--version requires a value}; shift 2 ;;
        --bin-dir) BIN_DIR=${2:?--bin-dir requires a value}; shift 2 ;;
        --systemd-dir) SYSTEMD_DIR=${2:?--systemd-dir requires a value}; shift 2 ;;
        --no-start) START_SERVICE=0; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
    esac
done

case "$REPO" in
    */* ) ;;
    * ) echo "Repository must be OWNER/REPOSITORY: $REPO" >&2; exit 2 ;;
esac
case "$VERSION" in
    *[!A-Za-z0-9._-]*|'') echo "Invalid release version: $VERSION" >&2; exit 2 ;;
esac
case "$BIN_DIR$SYSTEMD_DIR" in
    *' '*) echo "Install paths containing spaces are not supported by the systemd unit." >&2; exit 2 ;;
esac

if [ "$(uname -s)" != Linux ]; then
    echo "Shield's user service installer currently supports Linux only." >&2
    exit 1
fi

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64) GOARCH=amd64 ;;
    aarch64|arm64) GOARCH=arm64 ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

for command in curl tar awk install mktemp sha256sum; do
    command -v "$command" >/dev/null 2>&1 || {
        echo "Required command not found: $command" >&2
        exit 1
    }
done
if [ "$START_SERVICE" = 1 ]; then
    command -v systemctl >/dev/null 2>&1 || {
        echo "systemctl is required to enable the user service; use --no-start to skip." >&2
        exit 1
    }
fi

ARCHIVE="shield_${VERSION}_linux_${GOARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
UNIT_URL="https://raw.githubusercontent.com/${REPO}/${VERSION}/shield.service"
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/shield-install.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT HUP INT TERM

echo "Downloading Shield ${VERSION} for Linux/${GOARCH}..."
curl --fail --location --silent --show-error "$BASE_URL/$ARCHIVE" -o "$TMP_DIR/$ARCHIVE"
curl --fail --location --silent --show-error "$BASE_URL/sha256sums.txt" -o "$TMP_DIR/sha256sums.txt"
EXPECTED_SUM=$(awk -v file="$ARCHIVE" '$2 == file { print $1; exit }' "$TMP_DIR/sha256sums.txt")
if [ -z "$EXPECTED_SUM" ]; then
    echo "No checksum found for $ARCHIVE in the release." >&2
    exit 1
fi
printf '%s  %s\n' "$EXPECTED_SUM" "$TMP_DIR/$ARCHIVE" | sha256sum --check --status || {
    echo "Checksum verification failed for $ARCHIVE." >&2
    exit 1
}

mkdir -p "$TMP_DIR/unpacked" "$BIN_DIR" "$SYSTEMD_DIR"
tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR/unpacked"
[ -f "$TMP_DIR/unpacked/shield" ] && [ -f "$TMP_DIR/unpacked/shldd" ] || {
    echo "The release archive must contain shield and shldd." >&2
    exit 1
}
install -m 0755 "$TMP_DIR/unpacked/shield" "$BIN_DIR/shield"
install -m 0755 "$TMP_DIR/unpacked/shldd" "$BIN_DIR/shldd"

echo "Installing the systemd user unit..."
curl --fail --location --silent --show-error "$UNIT_URL" -o "$TMP_DIR/shield.service"
awk -v bindir="$BIN_DIR" '
    /^ExecStart=@SHIELD_BIN_DIR@\/shldd$/ { print "ExecStart=" bindir "/shldd"; next }
    { print }
' \
    "$TMP_DIR/shield.service" > "$TMP_DIR/shield.service.installed"
install -m 0644 "$TMP_DIR/shield.service.installed" "$SYSTEMD_DIR/shield.service"

if [ "$START_SERVICE" = 1 ]; then
    echo "Enabling and starting the Shield user service..."
    systemctl --user daemon-reload
    systemctl --user enable --now shield.service
fi

case ":${PATH:-}:" in
    *":$BIN_DIR:"*) ;;
    *) echo "Add $BIN_DIR to PATH to run 'shield' from your shell." ;;
esac
echo "Shield ${VERSION} installed in $BIN_DIR. Run: shield setup"
