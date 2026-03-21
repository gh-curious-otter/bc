#!/usr/bin/env bash
# Auto-deploy dogfood bcd when main branch has new commits.
# Usage:
#   scripts/auto-deploy.sh          # one-shot check and deploy
#   scripts/auto-deploy.sh --loop   # poll every 60s

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

deploy_if_needed() {
    git fetch origin main --quiet
    LOCAL=$(git rev-parse HEAD)
    REMOTE=$(git rev-parse origin/main)

    if [ "$LOCAL" = "$REMOTE" ]; then
        return 0
    fi

    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] New commits detected: $LOCAL -> $REMOTE"
    git pull origin main --quiet
    make deploy-dogfood
    echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] Deployed commit $(git rev-parse --short HEAD)"
}

if [ "${1:-}" = "--loop" ]; then
    echo "Auto-deploy loop started (polling every 60s)"
    while true; do
        deploy_if_needed || echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] Deploy failed, will retry"
        sleep 60
    done
else
    deploy_if_needed
fi
