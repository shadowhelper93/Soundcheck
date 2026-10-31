#!/usr/bin/env bash
# smoke-macos.sh — first-run install smoke test for Soundcheck on macOS.
# Builds the binary, starts an embedded server on a random port, curls the
# key API + UI endpoints, verifies error paths, then kills the server.
# Exits non-zero on any failure.
#
# Requires: Go >=1.21, curl. Run from the repo root.
set -euo pipefail

cd "$(dirname "$0")/.."
BIN=bin/soundcheck
PORT=$(( (RANDOM % 20000) + 20000 ))

fail() { echo "SMOKE FAIL: $1" >&2; exit 1; }

echo "[1/7] Building soundcheck..."
(cd ui && npm run build >/dev/null)
go build -o "$BIN" ./cmd/soundcheck || fail "go build"

echo "[2/7] Starting server on port $PORT..."
"$BIN" serve --port "$PORT" >/tmp/soundcheck-smoke.log 2>&1 &
SRV_PID=$!
trap 'kill $SRV_PID 2>/dev/null || true; wait $SRV_PID 2>/dev/null || true' EXIT

# Wait for /api/health with a 30s cap.
ok=0
for i in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:$PORT/api/health" >/dev/null 2>&1; then ok=1; break; fi
  sleep 1
done
[ "$ok" = 1 ] || { echo "--- server log ---"; cat /tmp/soundcheck-smoke.log; fail "server did not become healthy"; }

echo "[3/7] /api/health ................" ; curl -fsS "http://127.0.0.1:$PORT/api/health" && echo ok

echo "[4/7] /api/v1/doctor .............." ; curl -fsS "http://127.0.0.1:$PORT/api/v1/doctor" >/dev/null 2>&1 || fail "doctor endpoint"
curl -fsS "http://127.0.0.1:$PORT/api/v1/doctor" >/dev/null && echo ok

echo "[5/7] /api/v1/config ............." ; curl -fsS "http://127.0.0.1:$PORT/api/v1/config" >/dev/null 2>&1 || fail "config get"
curl -fsS "http://127.0.0.1:$PORT/api/v1/config" && echo ok

echo "[6/7] POST /api/v1/config/validate"
CODE=$(curl -s -o /tmp/soundcheck-smoke-validate.json -w '%{http_code}' -X POST "http://127.0.0.1:$PORT/api/v1/config/validate" -H 'Content-Type: application/json' -d '{}')
[ "$CODE" = "200" ] || fail "config/validate returned $CODE"
echo "200 ok"

echo "[7/7] bad PUT /api/v1/config (expect 400)"
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X PUT "http://127.0.0.1:$PORT/api/v1/config" -H 'Content-Type: application/json' -d '{"version":999}')
[ "$CODE" = "400" ] || fail "bad PUT returned $CODE"

echo "UI index:"
curl -fsS "http://127.0.0.1:$PORT/" | head -c 120 ; echo
code=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/")
[ "$code" = "200" ] || fail "UI returned $code"

echo
echo "SMOKE PASS: all endpoints verified on port $PORT"
