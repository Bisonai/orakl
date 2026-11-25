#!/usr/bin/env bash
# Verify a commit may be tagged and deployed:
#   1. it is on master (ancestor of, or equal to, MASTER_REF)
#   2. the `gate` job of its latest gate.yaml run (master push run preferred over
#      PR/dispatch runs) concluded `success`. A queued/in-progress run is polled
#      until done or TIMEOUT expires.
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

# gh api with 3 tries, so one transient API error doesn't abort a deploy.
api() {
  local i out
  for i in 1 2 3; do
    if out=$(gh api "$@"); then
      printf '%s\n' "$out"
      return 0
    fi
    echo "gh api failed (try $i/3)" >&2
    sleep 10
  done
  return 1
}

if ! git merge-base --is-ancestor "$sha" "$master_ref"; then
  echo "::error::$sha is not on master ($master_ref); only master commits may be tagged for deploy"
  exit 1
fi
echo "ok: $sha is on master ($master_ref)"

start=$(date +%s)
while true; do
  elapsed=$(( $(date +%s) - start ))
  # Latest gate.yaml run for this sha ("<id> <status>"); reruns keep the run id.
  run=$(api "repos/$repo/actions/runs?head_sha=$sha&per_page=100" \
    --jq '[.workflow_runs[] | select(.path == ".github/workflows/gate.yaml")] as $all
          | ([$all[] | select(.event == "push")] | if length > 0 then . else $all end)
          | sort_by(.id) | last
          | if . == null then "" else "\(.id) \(.status)" end')

  if [ -z "$run" ]; then
    if [ "$elapsed" -ge "$grace" ]; then
      echo "::error::no gate run found for $sha after ${elapsed}s (commit predates gate.yaml, or gate never ran for it); push a new master commit to deploy"
      exit 1
    fi
    echo "no gate run for $sha yet (${elapsed}s), waiting"
  else
    read -r run_id status <<<"$run"
    if [ "$status" = completed ]; then
      job=$(api "repos/$repo/actions/runs/$run_id/jobs" \
        --jq '[.jobs[] | select(.name == "gate")] | last
              | if . == null then "" else "\(.conclusion // "none") \(.html_url)" end')
      if [ -z "$job" ]; then
        echo "::error::gate run $run_id for $sha has no 'gate' job"
        exit 1
      fi
      read -r conclusion url <<<"$job"
      if [ "$conclusion" = success ]; then
        echo "ok: gate succeeded for $sha ($url)"
        exit 0
      fi
      echo "::error::gate for $sha concluded '$conclusion' ($url); fix master or rerun gate before tagging"
      exit 1
    fi
    if [ "$elapsed" -ge "$timeout" ]; then
      echo "::error::gate run $run_id for $sha still '$status' after ${elapsed}s"
      exit 1
    fi
    echo "gate run $run_id for $sha is '$status' (${elapsed}s), waiting"
  fi
  sleep "$interval"
done
