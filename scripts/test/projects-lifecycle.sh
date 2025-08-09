#!/usr/bin/env bash

# Projects Lifecycle Testing (REAL LIVE TESTS)
# WARNING: Creates real Atlas projects - use only in test environments

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; PURPLE='\033[0;35m'; NC='\033[0m'
print_info(){ echo -e "${PURPLE}ℹ $1${NC}"; }
print_success(){ echo -e "${GREEN}✓ $1${NC}"; }
print_warning(){ echo -e "${YELLOW}⚠ $1${NC}"; }
print_error(){ echo -e "${RED}✗ $1${NC}"; }

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

declare -a CREATED_PROJECTS=()

cleanup(){
  for ((i=${#CREATED_PROJECTS[@]}-1;i>=0;i--)); do
    pid="${CREATED_PROJECTS[i]}"
    "$PROJECT_ROOT/matlas" atlas projects delete "$pid" --yes 2>/dev/null || true
  done
}

main(){
  trap cleanup EXIT INT TERM
  if [[ -z "${ATLAS_PUB_KEY:-}" || -z "${ATLAS_API_KEY:-}" || -z "${ATLAS_ORG_ID:-}" ]]; then
    print_error "Missing ATLAS_PUB_KEY, ATLAS_API_KEY or ATLAS_ORG_ID"; exit 1; fi
  if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then (cd "$PROJECT_ROOT" && go build -o matlas) || { print_error "Build failed"; exit 1; }; fi

  local proj_name="live-proj-$(date +%s)"
  print_info "Creating project $proj_name in org $ATLAS_ORG_ID..."
  local out
  if ! out=$("$PROJECT_ROOT/matlas" atlas projects create "$proj_name" --org-id "$ATLAS_ORG_ID" 2>&1); then
    print_error "Project create failed"; echo "$out"
    # Graceful handling for organizations with region restrictions or policies that block creation
    if echo "$out" | grep -q "INVALID_REGION_RESTRICTION"; then
      print_warning "Organization policy prevents project creation. Skipping create/delete and validating visibility instead."
      if [[ -n "${ATLAS_PROJECT_ID:-}" ]]; then
        print_info "Verifying we can get existing project $ATLAS_PROJECT_ID and list by org..."
        "$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output yaml >/dev/null || { print_error "Get existing project failed"; exit 1; }
        "$PROJECT_ROOT/matlas" atlas projects list --org-id "$ATLAS_ORG_ID" --output json >/dev/null || print_warning "List by org failed"
        print_success "Visibility checks passed under restricted org"
        return 0
      else
        print_warning "ATLAS_PROJECT_ID not set; cannot perform visibility checks. Treating as skipped."
        return 0
      fi
    fi
    exit 1
  fi

  # Extract project ID from output if possible; otherwise, list and grep
  local pid
  pid=$("$PROJECT_ROOT/matlas" atlas projects list --output json 2>/dev/null | jq -r ".[] | select(.name==\"$proj_name\").id" 2>/dev/null || echo "")
  if [[ -z "$pid" || "$pid" == "null" ]]; then
    print_warning "Could not extract project ID from list; attempting fallback"
    pid=$(echo "$out" | grep -Eo '[a-f0-9]{24}' | head -1 || true)
  fi
  if [[ -z "$pid" ]]; then print_error "Failed to determine created project ID"; exit 1; fi
  CREATED_PROJECTS+=("$pid")
  print_success "Created project $proj_name ($pid)"

  print_info "Verifying get..."
  "$PROJECT_ROOT/matlas" atlas projects get --project-id "$pid" --output yaml >/dev/null || { print_error "Get failed"; exit 1; }
  print_success "Get succeeded"

  print_info "Listing projects in org (sanity)..."
  "$PROJECT_ROOT/matlas" atlas projects list --org-id "$ATLAS_ORG_ID" --output json >/dev/null || print_warning "List by org failed"

  print_info "Deleting project $pid..."
  if "$PROJECT_ROOT/matlas" atlas projects delete "$pid" --yes; then
    print_success "Deleted project $pid"
    CREATED_PROJECTS=() # already deleted
  else
    print_warning "Project delete failed; leaving for manual clean"
  fi
}

main "$@"


