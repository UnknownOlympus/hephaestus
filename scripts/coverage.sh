#!/usr/bin/env bash
set -euo pipefail

COV_MODE="${COV_MODE:-atomic}"
TMP_DIR=$(mktemp -d /tmp/gocov.XXXXXXXX)
COMBINED_FILE="${TMP_DIR}/summary.out"

collect_coverage() {
  local packages
  mapfile -t packages < <(go list ./... | grep -v '/mock\|/main\|/tasks')

  for pkg in "${packages[@]}"; do
    (
      local sanitized="${pkg//\//_}"
      local report="${TMP_DIR}/${sanitized}.cov"
      go test -covermode="${COV_MODE}" -coverprofile="${report}" "${pkg}"
    )
  done

  {
    echo "mode: ${COV_MODE}"
    grep -h -v "^mode:" "${TMP_DIR}"/*.cov
  } > "${COMBINED_FILE}"
}

collect_coverage
go tool cover -func="${COMBINED_FILE}"

if [[ "${1:-}" == "--html" ]]; then
  go tool cover -html="${COMBINED_FILE}"
fi
