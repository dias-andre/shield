#!/bin/sh

set -e
echo "=> Starting shield installation..."

ARCH=$(uname -m)
case "$ARCH" in
x86_64)
  GOARCH="amd64"
  ;;
aarch64 | arm64)
  GOARCH="arm64"
  ;;
*)
  echo "Architecture not supported: $ARCH"
  exit 1
  ;;
esac

VERSION="0.2.1"
REPO="dias-andre/shield"
BIN_DIR="$HOME/.local/bin"
SYSTEMD_DIR="$HOME/.config/systemd/user"
TAR_FILE="shield_${VERSION}_linux_${GOARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TAR_FILE}"
TEMP_DIR="/tmp/shield-bins"
# https://github.com/dias-andre/shield/archive/refs/tags/0.1.0.tar.gz
# https://github.com/dias-andre/shield/releases/download/0.1.0/shield_0.1.0_linux_amd64.tar.gz

echo "=> Downloading version ${VERSION}..."
curl -sSL "$DOWNLOAD_URL" -o "/tmp/${TAR_FILE}"

echo "=> Installing binaries in ${BIN_DIR}..."
tar -xzf "/tmp/${TAR_FILE}" -C "${TEMP_DIR}"
mv "${TEMP_DIR}/shield" "${BIN_DIR}/shield"
mv "${TEMP_DIR}/shldd" "${BIN_DIR}/shldd"
chmod +x "${BIN_DIR}/shield" "${BIN_DIR}/shldd"

SERVICE_URL="https://raw.githubusercontent.com/${REPO}/main/shield.service"
curl -sSL "$SERVICE_URL" -o "$SYSTEMD_DIR/shield.service"

sed -i "s|ExecStart=.*|ExecStart=${BIN_DIR}/shldd|" "${SYSTEMD_DIR}/shield.service"

echo "=> Starting shield service..."
systemctl --user daemon-reload
systemctl --user enable --now shield.service

rm -f "/tmp/${TAR_FILE}"
rm -rf "${TEMP_DIR}"

echo "Shield installed! Run 'shield ping' and 'shield setup' to start!"
