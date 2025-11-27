#!/bin/bash
#
# TodoList API CLI - Simple manual testing tool
# Usage: ./todo-cli.sh <command> [args]
#

set -e

CONFIG_FILE="$HOME/.todolist-cli.conf"
API_BASE="${API_BASE:-https://localhost:8443/api/v1}"

# Load config
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        source "$CONFIG_FILE"
    fi
}

# Save config
save_config() {
    cat > "$CONFIG_FILE" <<EOF
ACCESS_TOKEN="$ACCESS_TOKEN"
REFRESH_TOKEN="$REFRESH_TOKEN"
USER_EMAIL="$USER_EMAIL"
API_BASE="$API_BASE"
EOF
    chmod 600 "$CONFIG_FILE"
}

# Make API call
api() {
    local method="$1"
    local endpoint="$2"
    local data="$3"

    if [ -z "$data" ]; then
        curl -sk -X "$method" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -H "Content-Type: application/json" \
            "$API_BASE$endpoint" | jq .
    else
        curl -sk -X "$method" \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "$API_BASE$endpoint" | jq .
    fi
}

# Commands
cmd_login() {
    local email="${1:-$USER_EMAIL}"
    local password="$2"

    if [ -z "$email" ] || [ -z "$password" ]; then
        echo "Usage: $0 login <email> <password>"
        exit 1
    fi

    echo "Logging in as $email..."
    response=$(curl -sk -X POST "$API_BASE/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$email\",\"password\":\"$password\"}")

    ACCESS_TOKEN=$(echo "$response" | jq -r '.accessToken')
    REFRESH_TOKEN=$(echo "$response" | jq -r '.refreshToken')
    USER_EMAIL="$email"

    if [ "$ACCESS_TOKEN" = "null" ]; then
        echo "Login failed!"
        echo "$response" | jq .
        exit 1
    fi

    save_config
    echo "✓ Logged in successfully!"
    echo "Token saved to $CONFIG_FILE"
}

cmd_register() {
    local email="$1"
    local password="$2"
    local firstname="$3"
    local lastname="$4"

    if [ -z "$email" ] || [ -z "$password" ]; then
        echo "Usage: $0 register <email> <password> [firstname] [lastname]"
        exit 1
    fi

    echo "Registering $email..."
    curl -sk -X POST "$API_BASE/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$email\",\"password\":\"$password\",\"first_name\":\"$firstname\",\"last_name\":\"$lastname\"}" | jq .
}

cmd_profile() {
    api GET /auth/profile
}

cmd_lists() {
    api GET /lists
}

cmd_create_list() {
    local name="$1"
    local desc="$2"

    if [ -z "$name" ]; then
        echo "Usage: $0 create-list <name> [description]"
        exit 1
    fi

    api POST /lists "{\"name\":\"$name\",\"description\":\"$desc\"}"
}

cmd_get_list() {
    local list_id="$1"

    if [ -z "$list_id" ]; then
        echo "Usage: $0 get-list <list-id>"
        exit 1
    fi

    api GET "/lists/$list_id"
}

cmd_todos() {
    local list_id="$1"

    if [ -z "$list_id" ]; then
        echo "Usage: $0 todos <list-id>"
        exit 1
    fi

    api GET "/lists/$list_id/todos"
}

cmd_create_todo() {
    local list_id="$1"
    local title="$2"
    local desc="$3"

    if [ -z "$list_id" ] || [ -z "$title" ]; then
        echo "Usage: $0 create-todo <list-id> <title> [description]"
        exit 1
    fi

    api POST "/lists/$list_id/todos" "{\"title\":\"$title\",\"description\":\"$desc\"}"
}

cmd_health() {
    curl -sk "$API_BASE/../health/detailed" | jq .
}

cmd_config() {
    echo "Configuration:"
    echo "  Config file: $CONFIG_FILE"
    echo "  API Base: $API_BASE"
    echo "  User: $USER_EMAIL"
    echo "  Token: ${ACCESS_TOKEN:0:20}..."
}

cmd_help() {
    cat <<EOF
TodoList API CLI - Simple manual testing tool

USAGE:
    $0 <command> [args]

CONFIGURATION:
    API_BASE    Set API base URL (default: https://localhost:8443/api/v1)

    Example: API_BASE=https://your-server.example.com:8443/api/v1 $0 login user@test.com password

COMMANDS:
    Authentication:
        login <email> <password>           Login and save token
        register <email> <password> [fn] [ln]  Register new user
        profile                            Get current user profile

    Lists:
        lists                              Get all lists
        create-list <name> [desc]          Create a new list
        get-list <list-id>                 Get specific list

    Todos:
        todos <list-id>                    Get all todos in list
        create-todo <list-id> <title> [desc]  Create new todo

    Utilities:
        health                             Check API health
        config                             Show current config
        help                               Show this help

EXAMPLES:
    # Register and login
    $0 register test@example.com MyPass123 John Doe
    $0 login test@example.com MyPass123

    # Create a list and add todos
    $0 create-list "Shopping" "Weekly groceries"
    $0 create-todo <list-id> "Buy milk"
    $0 todos <list-id>

    # Check health
    $0 health

CONFIG FILE:
    Token and settings saved to: $CONFIG_FILE

EOF
}

# Main
load_config

case "${1:-help}" in
    login)          cmd_login "$2" "$3" ;;
    register)       cmd_register "$2" "$3" "$4" "$5" ;;
    profile)        cmd_profile ;;
    lists)          cmd_lists ;;
    create-list)    cmd_create_list "$2" "$3" ;;
    get-list)       cmd_get_list "$2" ;;
    todos)          cmd_todos "$2" ;;
    create-todo)    cmd_create_todo "$2" "$3" "$4" ;;
    health)         cmd_health ;;
    config)         cmd_config ;;
    help|--help|-h) cmd_help ;;
    *)              echo "Unknown command: $1"; cmd_help; exit 1 ;;
esac
