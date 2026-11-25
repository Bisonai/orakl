#!/usr/bin/env bash
# Verify a commit may be tagged and deployed:
#   1. it is on master (ancestor of, or equal to, MASTER_REF)
#   2. its latest `gate` check run (GitHub Actions) concluded `success`;
#      a queued/in-progress gate is polled until it completes or TIMEOUT expires.
#
# Usage: verify-deploy-commit.sh <sha>
# Env:   REPO (owner/name, default $GITHUB_REPOSITORY), MASTER_REF (default origin/master),
#        POLL_INTERVAL (s, default 30), TIMEOUT (s, default 1800),
#        MISSING_GRACE (s to wait for a gate run to appear, default 120). Needs git + gh.
# Dry-run locally: git fetch origin master; REPO=Bisonai/orakl .github/scripts/verify-deploy-commit.sh <sha>
set -euo pipefail

sha=$(git rev-parse --verify "${1:?usage: $0 <sha>}^{commit}")
repo=${REPO:-${GITHUB_REPOSITORY:?set REPO or GITHUB_REPOSITORY}}
master_ref=${MASTER_REF:-origin/master}
interval=${POLL_INTERVAL:-30}
timeout=${TIMEOUT:-1800}
grace=${MISSING_GRACE:-120}

if ! git merge-base --is-ancestor "$sha" "$master_ref"; then
  echo "::error::$sha is not on master ($master_ref); only master commits may be tagged for deploy"
  exit 1
fi
echo "ok: $sha is on master ($master_ref)"

start=$(date +%s)
while true; do
  # Latest `gate` check run from GitHub Actions (reruns create new runs; highest id wins).
  run=$(gh api "repos/$repo/commits/$sha/check-runs?check_name=gate&filter=all&per_page=100" \
    --jq '[.check_runs[] | select(.app.slug == "github-actions")] | sort_by(.id) | last
          | if . == null then "" else "\(.status) \(.conclusion // "-") \(.details_url)" end')
  elapsed=$(( $(date +%s) - start ))
  if [ -z "$run" ]; then
    if [ "$elapsed" -ge "$grace" ]; then
      echo "::error::no 'gate' check run found for $sha (after ${elapsed}s); commits before gate.yaml cannot be deployed"
      exit 1
    fi
    echo "no 'gate' check run for $sha yet (${elapsed}s), waiting"
  else
    read -r status conclusion url <<<"$run"
    if [ "$status" = completed ]; then
      if [ "$conclusion" = success ]; then
        echo "ok: gate succeeded for $sha ($url)"
        exit 0
      fi
      echo "::error::gate for $sha concluded '$conclusion' ($url); fix master or rerun gate before tagging"
      exit 1
    fi
    if [ "$elapsed" -ge "$timeout" ]; then
      echo "::error::gate for $sha still '$status' after ${elapsed}s ($url)"
      exit 1
    fi
    echo "gate for $sha is '$status' (${elapsed}s), waiting ($url)"
  fi
  sleep "$interval"
done
