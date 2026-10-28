#!/usr/bin/env bash
# Pulls the latest :latest image and recreates the container (only restarts when
# the image actually changed) so pushes to master roll out without a manual step.
# Migrations run on container start (MIGRATE_ON_START=true).
# Add to cron:
#   */5 * * * * /home/gewall/apps/dnd-forge-api/auto-pull.sh >> /home/gewall/apps/dnd-forge-api/autopull.log 2>&1
set -euo pipefail

cd "$(dirname "$0")"

COMPOSE_FILE="compose.yaml"
IMAGE="ghcr.io/gewall/dnd-forge-api:latest"

before="$(docker image inspect --format '{{.Id}}' "$IMAGE" 2>/dev/null || echo none)"
docker compose -f "$COMPOSE_FILE" pull api
after="$(docker image inspect --format '{{.Id}}' "$IMAGE" 2>/dev/null || echo none)"

if [ "$before" = "$after" ]; then
	echo "$(date -Is) no change"
	exit 0
fi

echo "$(date -Is) new image $after, deploying"
docker compose -f "$COMPOSE_FILE" up -d
docker image prune -f >/dev/null
