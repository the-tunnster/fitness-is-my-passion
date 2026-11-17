#!/usr/bin/env bash
# Test, cross-build, upload and restart the fitness tracker on the VM.
#
#   scripts/deploy.sh            deploy
#   scripts/deploy.sh --rollback put the previous binary back and restart
#
# Override the target with environment variables, e.g. DEPLOY_HOST=ubuntu@other scripts/deploy.sh
set -euo pipefail

HOST="${DEPLOY_HOST:-ubuntu@mumbai}"
DIR="${DEPLOY_DIR:-/home/ubuntu}"
SERVICE="${DEPLOY_SERVICE:-fitness-tracker}"
ARCH="${DEPLOY_ARCH:-arm64}"
PORT="${DEPLOY_PORT:-8090}"
BIN="fitness-tracker"

cd "$(dirname "$0")/.."

# Runs on the VM. The binary is never overwritten in place: Linux refuses to write to a
# running executable ("text file busy"), and a half-copied file would crash the service.
# Instead the new file is renamed over the old one, which is atomic; the running process
# keeps using the old file until the restart.
remote() { ssh "$HOST" "set -e; cd '$DIR'; $1"; }

restart_and_check() {
  remote "sudo -n systemctl restart '$SERVICE'"
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    if remote "curl -fsS -o /dev/null http://localhost:$PORT/" 2>/dev/null; then
      echo "OK: $SERVICE is up and answering on port $PORT."
      return 0
    fi
    sleep 1
  done
  echo "The service did not come up. Last log lines:" >&2
  remote "sudo -n journalctl -u '$SERVICE' -n 25 --no-pager" >&2 || true
  echo "Run scripts/deploy.sh --rollback to restore the previous version." >&2
  return 1
}

if [[ "${1:-}" == "--rollback" ]]; then
  remote "test -f $BIN.prev || { echo 'No previous binary to roll back to.' >&2; exit 1; }; mv -f $BIN.prev $BIN"
  restart_and_check
  exit 0
fi

echo "==> vet and test"
go vet ./...
go test ./...

echo "==> build linux/$ARCH"
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath -ldflags="-s -w" -o "dist/$BIN" .

echo "==> upload to $HOST:$DIR"
scp -q "dist/$BIN" "$HOST:$DIR/$BIN.new"

echo "==> install and restart"
remote "chmod +x $BIN.new; [ -f $BIN ] && cp -p $BIN $BIN.prev; mv -f $BIN.new $BIN"
restart_and_check
