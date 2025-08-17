#!/usr/bin/env bash

# Network Access Lifecycle Testing (REAL LIVE TESTS)
# WARNING: Modifies network access list - use only in test environments

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; PURPLE='\033[0;35m'; NC='\033[0m'
print_info(){ echo -e "${PURPLE}ℹ $1${NC}"; }
print_success(){ echo -e "${GREEN}✓ $1${NC}"; }
print_warning(){ echo -e "${YELLOW}⚠ $1${NC}"; }
print_error(){ echo -e "${RED}✗ $1${NC}"; }

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/network-lifecycle"
mkdir -p "$TEST_REPORTS_DIR"

declare -a CREATED_IPS=()

track_ip(){ CREATED_IPS+=("$1"); print_info "Tracking network entry: $1"; }

cleanup(){
  print_info "Cleaning up network entries..."
  for ((i=${#CREATED_IPS[@]}-1;i>=0;i--)); do
    ip="${CREATED_IPS[i]}"
    "$PROJECT_ROOT/matlas" atlas network delete "$ip" --project-id "$ATLAS_PROJECT_ID" --yes 2>/dev/null || true
  done
}

check_env(){
  if [[ -z "${ATLAS_PUB_KEY:-}" || -z "${ATLAS_API_KEY:-}" || -z "${ATLAS_PROJECT_ID:-}" ]]; then
    print_error "Missing ATLAS_* env vars"; return 1; fi
  if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
    (cd "$PROJECT_ROOT" && go build -o matlas) || { print_error "Build failed"; return 1; }
  fi
}

test_cli_network_lifecycle(){
  print_info "CLI network create/list/delete..."
  local ip="203.0.113.$((RANDOM%200+1))"
  if "$PROJECT_ROOT/matlas" atlas network create --project-id "$ATLAS_PROJECT_ID" --ip-address "$ip" --comment "network-live" 2>/dev/null; then
    track_ip "$ip"; print_success "Created $ip"
  else
    print_error "Network create failed"; return 1
  fi
  sleep 2
  if "$PROJECT_ROOT/matlas" atlas network list --project-id "$ATLAS_PROJECT_ID" | grep -q "$ip"; then
    print_success "Entry visible in list"
  else
    print_warning "Entry not visible yet"
  fi
  "$PROJECT_ROOT/matlas" atlas network delete "$ip" --project-id "$ATLAS_PROJECT_ID" --yes 2>/dev/null || print_warning "Delete failed"
}

test_yaml_network_apply_destroy(){
  print_info "YAML network apply/destroy..."
  local ip="198.51.100.$((RANDOM%200+1))"
  local cfg="$TEST_REPORTS_DIR/network.yaml"
  local project_name
  project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null || echo "$ATLAS_PROJECT_ID")
  cat > "$cfg" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata: { name: network-yaml }
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: NetworkAccess
    metadata: { name: test-ip }
    spec:
      projectName: "$project_name"
      ipAddress: $ip
      comment: yaml-network-test
EOF
  "$PROJECT_ROOT/matlas" infra validate -f "$cfg" || { print_error "Validate failed"; return 1; }
  "$PROJECT_ROOT/matlas" infra apply -f "$cfg" --project-id "$ATLAS_PROJECT_ID" --preserve-existing --auto-approve || { print_error "Apply failed"; return 1; }
  track_ip "$ip"
  sleep 2
  "$PROJECT_ROOT/matlas" infra destroy -f "$cfg" --project-id "$ATLAS_PROJECT_ID" --target network-access --auto-approve || print_warning "Destroy failed"
}

main(){
  trap cleanup EXIT INT TERM
  check_env || exit 1
  local failures=0
  test_cli_network_lifecycle || ((failures++))
  test_yaml_network_apply_destroy || ((failures++))
  if [[ $failures -eq 0 ]]; then
    print_success "Network lifecycle tests passed"
  else
    print_error "$failures network lifecycle test(s) failed"; exit 1
  fi
}

main "$@"


