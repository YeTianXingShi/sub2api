#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

# Go module/import paths, the license, and DEV_GUIDE's explicitly labelled
# upstream attribution intentionally remain on Wei-Shaw/sub2api.
targets=(.github deploy docs frontend/src README.md README_CN.md README_JA.md Dockerfile Dockerfile.goreleaser)
if rg -ni 'Wei-Shaw/sub2api|Wei-Shaw%2Fsub2api|weishaw/sub2api' "${targets[@]}"; then
  echo "Operational repository reference points back to upstream" >&2
  exit 1
fi

rg -q 'GITHUB_REPO="YeTianXingShi/sub2api"' deploy/install.sh
rg -q "grep -E '\^v-custom" deploy/install.sh
rg -Fq 'archive_version="${GITHUB_REF_NAME#v-}"' .github/workflows/custom-release.yml
rg -q 'ghcr.io/yetianxingshi/sub2api' .github/workflows/custom-release.yml
rg -q 'githubRepo[[:space:]]*= "YeTianXingShi/sub2api"' backend/internal/service/update_service.go
echo "Custom repository audit passed"
