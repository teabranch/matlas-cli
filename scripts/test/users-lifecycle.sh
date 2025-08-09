#!/usr/bin/env bash

# Users Lifecycle Testing for matlas-cli (REAL LIVE TESTS)
# WARNING: Creates real Atlas users - use only in test environments

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_REPORTS_DIR="$PROJECT_ROOT/test-reports/users-lifecycle"

declare -a CREATED_USERS=()

print_info(){ echo -e "${PURPLE}ℹ $1${NC}"; }
print_success(){ echo -e "${GREEN}✓ $1${NC}"; }
print_warning(){ echo -e "${YELLOW}⚠ $1${NC}"; }
print_error(){ echo -e "${RED}✗ $1${NC}"; }

track_user(){ CREATED_USERS+=("$1"); print_info "Tracking user: $1"; }

check_environment(){
  mkdir -p "$TEST_REPORTS_DIR"
  if [[ -z "${ATLAS_PUB_KEY:-}" || -z "${ATLAS_API_KEY:-}" || -z "${ATLAS_PROJECT_ID:-}" ]]; then
    print_error "Missing ATLAS_PUB_KEY, ATLAS_API_KEY or ATLAS_PROJECT_ID"
    return 1
  fi
  if [[ ! -f "$PROJECT_ROOT/matlas" ]]; then
    print_info "Building matlas binary..."
    (cd "$PROJECT_ROOT" && go build -o matlas) || { print_error "Build failed"; return 1; }
  fi
  print_success "Environment ready"
}

cleanup(){
  print_info "Cleaning up users..."
  for ((i=${#CREATED_USERS[@]}-1;i>=0;i--)); do
    u="${CREATED_USERS[i]}"
    "$PROJECT_ROOT/matlas" atlas users delete "$u" --project-id "$ATLAS_PROJECT_ID" --database-name admin --yes 2>/dev/null || true
  done
}

test_cli_users_lifecycle(){
  print_info "CLI users lifecycle..."
  local uname="live-user-$(date +%s)"
  if "$PROJECT_ROOT/matlas" atlas users create --project-id "$ATLAS_PROJECT_ID" --username "$uname" --database-name admin --roles read@admin --password "LiveUserInit123!" 2>/dev/null; then
    track_user "$uname"; print_success "Created user $uname"
  else
    print_error "Create user failed"; return 1
  fi

  sleep 2
  if "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$uname"; then
    print_success "User appears in list"
  else
    print_warning "User not visible yet"
  fi

  # Update password
  "$PROJECT_ROOT/matlas" atlas users update "$uname" --project-id "$ATLAS_PROJECT_ID" --database-name admin --password "LiveUserNew456!" 2>/dev/null || { print_error "Password update failed"; return 1; }
  # Update roles
  "$PROJECT_ROOT/matlas" atlas users update "$uname" --project-id "$ATLAS_PROJECT_ID" --database-name admin --roles readWrite@admin 2>/dev/null || { print_error "Role update failed"; return 1; }

  print_success "CLI users lifecycle completed"
}

test_yaml_users_apply_destroy(){
  print_info "YAML users apply/destroy..."
  local uname="yaml-user-$(date +%s)"
  local cfg="$TEST_REPORTS_DIR/users.yaml"
  local project_name
  project_name=$("$PROJECT_ROOT/matlas" atlas projects get --project-id "$ATLAS_PROJECT_ID" --output json 2>/dev/null | jq -r '.name' 2>/dev/null || echo "$ATLAS_PROJECT_ID")
  cat > "$cfg" << EOF
apiVersion: matlas.mongodb.com/v1
kind: ApplyDocument
metadata: { name: users-yaml }
resources:
  - apiVersion: matlas.mongodb.com/v1
    kind: DatabaseUser
    metadata: { name: $uname }
    spec:
      projectName: "$project_name"
      username: $uname
      databaseName: admin
      password: UsersYamlPass123!
      roles:
        - roleName: read
          databaseName: admin
EOF
  "$PROJECT_ROOT/matlas" infra validate -f "$cfg" || { print_error "Validate failed"; return 1; }
  "$PROJECT_ROOT/matlas" infra -f "$cfg" --project-id "$ATLAS_PROJECT_ID" --auto-approve || { print_error "Apply failed"; return 1; }
  track_user "$uname"
  sleep 3
  "$PROJECT_ROOT/matlas" infra destroy -f "$cfg" --project-id "$ATLAS_PROJECT_ID" --auto-approve || print_warning "Destroy failed"
  sleep 3
  if ! "$PROJECT_ROOT/matlas" atlas users list --project-id "$ATLAS_PROJECT_ID" | grep -q "$uname"; then
    print_success "YAML user cleaned up"
  else
    print_warning "YAML user may still be cleaning up"
  fi
}

main(){
  trap cleanup EXIT INT TERM
  check_environment || exit 1
  local failures=0
  test_cli_users_lifecycle || ((failures++))
  test_yaml_users_apply_destroy || ((failures++))
  if [[ $failures -eq 0 ]]; then
    print_success "Users lifecycle tests passed"
  else
    print_error "$failures users lifecycle test(s) failed"; exit 1
  fi
}

main "$@"


