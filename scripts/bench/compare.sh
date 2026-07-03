#!/usr/bin/env bash
# Helix VPN — Benchmark comparison tool
#
# Purpose:  Compare two benchmark CSV files side-by-side and compute the
#           percentage change for each metric.
# Usage:    ./scripts/bench/compare.sh <reference.csv> <candidate.csv>
# Inputs:   Two benchmark CSV files in the format produced by run.sh
# Outputs:  Side-by-side diff table with percentage change
# Side-effects: none
# Dependencies: bash 4+, standard POSIX tools (awk, column, sort)

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <reference.csv> <candidate.csv>"
  echo ""
  echo "Compare two benchmark CSV files and show side-by-side diffs."
  echo ""
  echo "Examples:"
  echo "  $0 bench-results/bench-001.csv bench-results/bench-002.csv"
  echo "  $0 --last  # compare the two most recent CSV files"
  exit 1
fi

# Special mode: compare the two most recent CSV files
if [[ "$1" == "--last" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
  SEARCH_DIR="${SCRIPT_DIR}/../../bench-results"
  if [[ ! -d "${SEARCH_DIR}" ]]; then
    echo "ERROR: --last mode requires ${SEARCH_DIR}/ to exist"
    exit 1
  fi
  mapfile -t files < <(find "${SEARCH_DIR}" -name 'bench-*.csv' -printf '%T@ %p\n' 2>/dev/null | sort -rn | awk '{print $2}')
  if [[ ${#files[@]} -lt 2 ]]; then
    echo "ERROR: Need at least 2 CSV files in ${SEARCH_DIR} for --last mode (found ${#files[@]})"
    exit 1
  fi
  REF="${files[0]}"
  CAND="${files[1]}"
  echo "Comparing newest against previous:"
  echo "  Candidate:  ${REF}"
  echo "  Reference:  ${CAND}"
  echo ""
else
  REF="$1"
  CAND="$2"
fi

# Validate both files exist
if [[ ! -f "${REF}" ]]; then
  echo "ERROR: Reference file not found: ${REF}"
  exit 1
fi
if [[ ! -f "${CAND}" ]]; then
  echo "ERROR: Candidate file not found: ${CAND}"
  exit 1
fi

# ---------------------------------------------------------------------------
# Build a lookup: test_type,metric => value
# ---------------------------------------------------------------------------
declare -A ref_map cand_map

build_map() {
  local -n map="$1"
  local file="$2"
  while IFS=, read -r ts tt metric value unit; do
    # Skip header
    [[ "${tt}" == "test_type" ]] && continue
    local key="${tt},${metric}"
    map["${key}"]="${value}:${unit}"
  done < "${file}"
}

build_map ref_map  "${REF}"
build_map cand_map "${CAND}"

# ---------------------------------------------------------------------------
# Gather all unique keys
# ---------------------------------------------------------------------------
all_keys=$( (echo "${!ref_map[@]}" "${!cand_map[@]}") | tr ' ' '\n' | sort -u)

echo "=== Benchmark Comparison ==="
echo "Reference:  ${REF}"
echo "Candidate:  ${CAND}"
echo ""

printf "%-35s %-15s %-15s %-10s\n" "Metric" "Reference" "Candidate" "Change"
printf "%-35s %-15s %-15s %-10s\n" "------" "---------" "---------" "------"

changed=false
while IFS= read -r key; do
  [[ -z "${key}" ]] && continue
  local ref_val="${ref_map[$key]:-}"
  local cand_val="${cand_map[$key]:-}"

  local ref_display cand_display change_display

  if [[ -n "${ref_val}" ]]; then
    val="${ref_val%%:*}"
    unit="${ref_val##*:}"
    ref_display="${val} ${unit}"
  else
    ref_display="(absent)"
  fi

  if [[ -n "${cand_val}" ]]; then
    val="${cand_val%%:*}"
    unit="${cand_val##*:}"
    cand_display="${val} ${unit}"
  else
    cand_display="(absent)"
  fi

  # Compute percentage change
  change_display="-"
  if [[ -n "${ref_map[$key]:-}" && -n "${cand_map[$key]:-}" ]]; then
    r="${ref_map[$key]%%:*}"
    c="${cand_map[$key]%%:*}"
    if [[ "${r}" != "FAIL" && "${r}" != "SKIP" && "${c}" != "FAIL" && "${c}" != "SKIP" ]]; then
      # Guard against zero
      if echo "${r} != 0" | bc -l 2>/dev/null | grep -q 1; then
        local pct
        pct=$(echo "scale=2; (${c} - ${r}) / ${r} * 100" | bc -l)
        if echo "${pct} < 0" | bc -l | grep -q 1; then
          change_display="${pct}% (better)"
        elif echo "${pct} > 0" | bc -l | grep -q 1; then
          change_display="+${pct}% (worse)"
        else
          change_display="0.00%"
        fi
      fi
    fi
  fi

  # Only show metrics that are present in at least one file
  if [[ -n "${ref_val}" || -n "${cand_val}" ]]; then
    changed=true
    local key_display
    key_display="${key//,/ }"
    printf "%-35s %-15s %-15s %-10s\n" "${key_display}: " "${ref_display}" "${cand_display}" "${change_display}"
  fi
done <<< "${all_keys}"

echo ""

if ! $changed; then
  echo "No overlapping metrics found between the two files."
fi

echo "=== Comparison complete ==="
