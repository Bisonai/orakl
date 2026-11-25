#!/usr/bin/env bash
# Verify a commit may be tagged and deployed:
#   1. it is on master (ancestor of, or equal to, MASTER_REF)
#   2. its latest `gate` check run (GitHub Actions) concluded `success`.
#      While a gate workflow run for the commit is still queued/in progress it is
#      polled until done or TIMEOUT expires (the `gate` job's check run only appears
#      once node/node-services finish).
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
  elapsed=$(( $(date +%s) - start ))
  # gate workflow runs for this commit: "<total> <not yet completed>"
  runs=$(gh api "repos/$repo/actions/runs?head_sha=$sha&per_page=100" \
    --jq '[.workflow_runs[] | select(.path == ".github/workflows/gate.yaml")]
          | "\(length) \(map(select(.status != "completed")) | length)"')
  read -r total pending <<<"$runs"

  if [ "$total" -eq 0 ] && [ "$elapsed" -lt "$grace" ]; then
    echo "no gate run for $sha yet (${elapsed}s), waiting"
  elif [ "$pending" -gt 0 ]; then
    if [ "$elapsed" -ge "$timeout" ]; then
      echo "::error::gate for $sha still not completed after ${elapsed}s"
      exit 1
    fi
    echo "gate for $sha still running (${elapsed}s), waiting"
  else
    # Latest `gate` check run from GitHub Actions (reruns create new ones; highest id wins).
    run=$(gh api "repos/$repo/commits/$sha/check-runs?check_name=gate&filter=all&per_page=100" \
      --jq '[.check_runs[] | select(.app.slug == "github-actions")] | sort_by(.id) | last
            | if . == null then "" else "\(.conclusion // "none") \(.details_url)" end')
    if [ -z "$run" ]; then
      echo "::error::no 'gate' check run found for $sha ($total gate workflow run(s)); commits before gate.yaml cannot be deployed"
      exit 1
    fi
    read -r conclusion url <<<"$run"
    if [ "$conclusion" = success ]; then
      echo "ok: gate succeeded for $sha ($url)"
      exit 0
    fi
    echo "::error::gate for $sha concluded '$conclusion' ($url); fix master or rerun gate before tagging"
    exit 1
  fi
  sleep "$interval"
done
