#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${1:-/opt/buildwebs233}"
REPO="${2:-neko233-com/buildwebs233}"
VERSION="${3:-latest}"
SERVICE_NAME="${4:-buildwebs233-server}"

if [[ "$(id -u)" -eq 0 ]]; then
  echo "[buildwebs233] 不要用 root 运行安装脚本。请使用普通用户并确保有目录写权限。"
fi

log() {
  echo "[buildwebs233] $1"
}

mkdir -p "$INSTALL_DIR"

if [[ "$VERSION" != "latest" ]] && [[ "$VERSION" != v* ]]; then
  VERSION="v$VERSION"
fi

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*\"tag_name\": \"\\([^\"]*\\)\".*/\\1/p' | head -1)"
fi

if [[ -z "${VERSION}" ]]; then
  VERSION="latest"
fi

ASSET_URL="https://github.com/${REPO}/releases/download/${VERSION}/buildwebs233-server-linux-amd64.tar.gz"
TMPDIR="$(mktemp -d)"

if command -v curl >/dev/null 2>&1; then
  if curl -fsSL -o "$TMPDIR/buildwebs233.tar.gz" "$ASSET_URL"; then
    tar -xzf "$TMPDIR/buildwebs233.tar.gz" -C "$TMPDIR"
    install -m 0755 "$TMPDIR/buildwebs233-server" "$INSTALL_DIR/buildwebs233-server"
    log "installed prebuilt binary"
  else
    log "download failed, fallback to local source build"
    if command -v go >/dev/null 2>&1; then
      go build -o "$INSTALL_DIR/buildwebs233-server" ./cmd/buildwebs233-server
    else
      log "go not found and no prebuilt asset available"
      exit 1
    fi
  fi
else
  echo "curl required"
  exit 1
fi

mkdir -p "$INSTALL_DIR/web" "$INSTALL_DIR/data"
cp -f "$(dirname "$0")/../server.yaml" "$INSTALL_DIR/server.yaml"

cat > "$INSTALL_DIR/service.conf" <<EOF
[Unit]
Description=BuildWebS233 Server
After=network.target

[Service]
ExecStart=$INSTALL_DIR/buildwebs233-server -config $INSTALL_DIR/server.yaml
Restart=on-failure
RestartSec=2

[Install]
WantedBy=multi-user.target
EOF

log "installed at $INSTALL_DIR"
log "start with: $INSTALL_DIR/buildwebs233-server -config $INSTALL_DIR/server.yaml"

rm -rf "$TMPDIR"
