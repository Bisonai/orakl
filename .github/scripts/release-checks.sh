#!/usr/bin/env bash
# Pre-tag checks for release.yaml. Fails unless all hold:
#   1. REF is refs/heads/master (the workflow was dispatched on master)
#   2. <version> is X.Y.Z and <service>/v<version> matches the tag+deploy.yaml tag pattern
#   3. tag <service>/v<version> does not exist on the remote
#   4. <version> is greater than the service's latest remote <service>/v* tag
#   5. verify-deploy-commit.sh passes for <sha> (on master, `gate` green; waits for a running gate)
# On success prints "tag=<tag>" to stdout and, if set, to $GITHUB_OUTPUT.
#
# Usage: release-checks.sh <service> <version> <sha>
# Env:   REF (default $GITHUB_REF), REMOTE (git remote, default origin),
#        REPO / MASTER_REF / POLL_INTERVAL / TIMEOUT / MISSING_GRACE (passed to verify-deploy-commit.sh).
# Dry-run locally: git fetch origin master; REF=refs/heads/master REPO=Bisonai/orakl \
#   .github/scripts/release-checks.sh node 0.0.53 "$(git rev-parse origin/master)"
set -euo pipefail

service=${1:?usage: $0 <service> <version> <sha>}
version=${2:?usage: $0 <service> <version> <sha>}
sha=${3:?usage: $0 <service> <version> <sha>}
ref=${REF:-${GITHUB_REF:-}}
remote=${REMOTE:-origin}
tag="${service}/v${version}"

if [ "$ref" != refs/heads/master ]; then
  echo "::error::release must run on master (got ref '${ref:-<unset>}'); dispatch with --ref master"
  exit 1
fi
echo "ok: ref is $ref"

if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "::error::version '$version' is not X.Y.Z (no 'v' prefix, no suffix)"
  exit 1
fi
# Same pattern tag+deploy.yaml parses; a tag it can't parse would fail there after the push.
if [[ ! $tag =~ ^([a-zA-Z0-9]+-?[a-zA-Z0-9]+)/v([0-9]+\.[0-9]+\.[0-9]+)$ ]]; then
  echo "::error::tag '$tag' does not match <service>/vX.Y.Z as parsed by tag+deploy.yaml"
  exit 1
fi
echo "ok: tag $tag is well-formed"

# Remote is the source of truth (local tags may be stale or missing).
remote_tags=$(git ls-remote --tags --refs "$remote" "refs/tags/${service}/v*") \
  || { echo "::error::git ls-remote $remote failed"; exit 1; }
if grep -qxF "refs/tags/$tag" <<<"$(cut -f2 <<<"$remote_tags")"; then
  echo "::error::tag $tag already exists on $remote"
  exit 1
fi
echo "ok: $tag does not exist on $remote"

latest=$(cut -f2 <<<"$remote_tags" | sed -n "s,^refs/tags/${service}/v\([0-9]*\.[0-9]*\.[0-9]*\)$,\1,p" | sort -V | tail -n1)
if [ -n "$latest" ]; then
  highest=$(printf '%s\n%s\n' "$latest" "$version" | sort -V | tail -n1)
  if [ "$highest" != "$version" ] || [ "$latest" = "$version" ]; then
    echo "::error::version $version is not greater than latest ${service}/v${latest}"
    exit 1
  fi
  echo "ok: $version > latest ${service}/v${latest}"
else
  echo "ok: no previous ${service}/v* tag; $version is the first"
fi

"$(dirname "$0")/verify-deploy-commit.sh" "$sha"

echo "tag=$tag"
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "tag=$tag" >> "$GITHUB_OUTPUT"
fi
