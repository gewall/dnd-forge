#!/usr/bin/env bash
# Registers, tests and activates an AI provider without printing any secret.
# Run it interactively on the VPS:
#   ssh -t vps ~/apps/dnd-forge-api/setup-ai-provider.sh
# Needs an ADMIN account (docker compose exec api /app/tools seed-admin -email ...).
set -euo pipefail
cd "$(dirname "$0")"

BASE="http://127.0.0.1:${APP_PORT:-8092}"
INTERNAL_KEY=$(grep '^INTERNAL_API_KEY=' .env | cut -d= -f2-)

# api METHOD PATH [TOKEN]: sends the JSON on stdin; prints the body and
# fails on a non-2xx status.
api() {
	local args=(-sS -X "$1" "$BASE$2" -H "X-Internal-Key: $INTERNAL_KEY" -H 'Content-Type: application/json'
		-w '\n%{http_code}')
	[ -n "${3:-}" ] && args+=(-H "Authorization: Bearer $3")
	[ "$1" != GET ] && args+=(--data-binary @-)
	local out code
	out=$(curl "${args[@]}")
	code=${out##*$'\n'}
	out=${out%$'\n'*}
	if [ "${code:0:1}" != 2 ]; then
		echo "gagal ($code): $(jq -r '.detail // .' <<<"$out" 2>/dev/null || echo "$out")" >&2
		return 1
	fi
	printf '%s' "$out"
}

read -rp "Email admin: " email
read -rsp "Password admin: " password
echo
token=$(jq -n --arg e "$email" --arg p "$password" '{email: $e, password: $p}' | api POST /v1/auth/login | jq -r .accessToken)
unset password

read -rp "Jenis provider (deepseek/openai) [deepseek]: " type
type=${type:-deepseek}
case "$type" in
	deepseek) capability=text default_model=deepseek-v4-flash models_url=https://api.deepseek.com/models ;;
	openai) capability=image default_model=gpt-image-1 models_url=https://api.openai.com/v1/models ;;
	*) echo "jenis tidak dikenal: $type" >&2; exit 1 ;;
esac
read -rsp "API key $type: " apikey
echo

echo "Model yang tersedia untuk key ini:"
if ! models=$(printf 'Authorization: Bearer %s\n' "$apikey" | curl -sS -H @- "$models_url" | jq -er '.data[].id' 2>/dev/null); then
	models="(tidak bisa mengambil daftar model; key mungkin salah)"
fi
sed 's/^/  - /' <<<"$models"
read -rp "Model [$default_model]: " model
model=${model:-$default_model}
read -rp "Label [$type $model]: " label
label=${label:-"$type $model"}

id=$(jq -n --arg t "$type" --arg c "$capability" --arg l "$label" --arg m "$model" --arg k "$apikey" \
	'{type: $t, capability: $c, label: $l, defaultModel: $m, apiKey: $k}' |
	api POST /v1/admin/ai/providers "$token" | jq -r .id)
unset apikey
echo "provider dibuat: $id"

if ! api POST "/v1/admin/ai/providers/$id/test" "$token" </dev/null >/dev/null; then
	api DELETE "/v1/admin/ai/providers/$id" "$token" </dev/null >/dev/null || true
	echo "provider dihapus lagi; periksa key lalu jalankan ulang" >&2
	exit 1
fi
echo "uji koneksi: berhasil"
api POST "/v1/admin/ai/providers/$id/activate" "$token" </dev/null >/dev/null
echo "aktif untuk kemampuan: $capability"
