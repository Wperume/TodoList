# TodoList CLI - User Guide

A type-safe, feature-complete command-line client for the TodoList API built with Go and Cobra.

## Features

- **Type-safe**: Uses the same models as the API server
- **Secure**: Stores credentials with file permissions 0600
- **Automatic token management**: Saves and reuses access/refresh tokens
- **Full API coverage**: All authentication, list, and todo endpoints
- **Flexible output**: Human-readable tables or JSON output
- **Self-signed cert support**: `--insecure` flag for development

## Installation

### Build from Source

```bash
# From the project root
go build -o todocli cmd/todocli/*.go

# Optionally, move to your PATH
sudo mv todocli /usr/local/bin/
```

### Install with go install

```bash
go install todolist-api/cmd/todocli@latest
```

## Configuration

Configuration is stored in `~/.todolist-cli.json` with the following structure:

```json
{
  "apiBaseUrl": "https://192.18.159.108:8443/api/v1",
  "accessToken": "eyJhbGc...",
  "refreshToken": "eyJhbGc...",
  "userEmail": "user@example.com",
  "insecureSkipVerify": true
}
```

### Configuration Commands

```bash
# View current configuration
todocli config show

# Set API URL
todocli config set --api-url https://192.18.159.108:8443/api/v1

# Enable insecure mode (skip TLS verification)
todocli config set --insecure

# Get config file path
todocli config path
```

## Global Flags

All commands support these global flags:

- `--api-url <url>`: Override API base URL
- `--insecure`: Skip TLS certificate verification
- `--json`: Output raw JSON responses

## Authentication

### Register a New Account

```bash
# Interactive password prompts
todocli auth register user@example.com

# With name
todocli auth register user@example.com --first-name John --last-name Doe
```

### Login

```bash
# Interactive password prompt
todocli auth login user@example.com

# If email is saved in config
todocli auth login
```

Tokens are automatically saved to config after successful login.

### View Profile

```bash
todocli auth profile
```

### Update Profile

```bash
todocli auth update-profile --first-name Jane --last-name Smith
```

### Change Password

```bash
# Interactive prompts for current and new password
todocli auth change-password
```

### Logout

```bash
todocli auth logout
```

## List Management

### List All Lists

```bash
# Table format
todocli list ls

# JSON format
todocli list ls --json
```

Output:
```
ID                                    NAME       DESCRIPTION         TODOS  CREATED
f47ac10b-58cc-4372-a567-0e02b2c3d479  Shopping   Weekly groceries    5      2025-01-15
```

### Get a Specific List

```bash
todocli list get <list-id>
```

### Create a New List

```bash
# With name only
todocli list create "Shopping List"

# With description
todocli list create "Shopping List" --description "Weekly groceries"
todocli list create "Shopping List" -d "Weekly groceries"  # Short form
```

### Update a List

```bash
# Update name
todocli list update <list-id> --name "New Name"

# Update description
todocli list update <list-id> --description "New description"

# Update both
todocli list update <list-id> -n "New Name" -d "New description"
```

### Delete a List

```bash
# With confirmation prompt
todocli list delete <list-id>

# Skip confirmation
todocli list delete <list-id> --yes
todocli list delete <list-id> -y  # Short form
```

## Todo Management

### List All Todos in a List

```bash
# Table format
todocli todo ls <list-id>

# JSON format
todocli todo ls <list-id> --json
```

Output:
```
ID                                    DESCRIPTION           PRIORITY  DUE DATE    COMPLETED
a1b2c3d4-...                         Buy milk              high      2025-01-20  [✓]
e5f6g7h8-...                         Pick up laundry       medium    -           [ ]
```

### Get a Specific Todo

```bash
todocli todo get <list-id> <todo-id>
```

### Create a Todo

```bash
# Basic todo
todocli todo create <list-id> "Buy milk"

# With priority
todocli todo create <list-id> "Buy milk" --priority high
todocli todo create <list-id> "Buy milk" -p high  # Short form

# With due date
todocli todo create <list-id> "Buy milk" --due-date 2025-01-20
todocli todo create <list-id> "Buy milk" -d 2025-01-20  # Short form

# With both
todocli todo create <list-id> "Buy milk" -p high -d 2025-01-20
```

Priority values: `low`, `medium`, `high`

### Update a Todo

```bash
# Update description
todocli todo update <list-id> <todo-id> --description "New description"

# Update priority
todocli todo update <list-id> <todo-id> --priority low

# Update due date
todocli todo update <list-id> <todo-id> --due-date 2025-01-25

# Mark as completed
todocli todo update <list-id> <todo-id> --completed

# Mark as incomplete
todocli todo update <list-id> <todo-id> --incomplete

# Update multiple fields
todocli todo update <list-id> <todo-id> -p high -d 2025-01-25 --completed
```

### Delete a Todo

```bash
# With confirmation prompt
todocli todo delete <list-id> <todo-id>

# Skip confirmation
todocli todo delete <list-id> <todo-id> --yes
todocli todo delete <list-id> <todo-id> -y  # Short form
```

## Example Workflows

### Complete Workflow Example

```bash
# 1. Register and login
todocli auth register test@example.com --first-name Test --last-name User
# Password: ********
# ✓ Registration successful!

# 2. Create a shopping list
todocli list create "Shopping" -d "Weekly groceries"
# ✓ List created successfully
# ID:   f47ac10b-58cc-4372-a567-0e02b2c3d479

# 3. Add some todos
todocli todo create f47ac10b-58cc-4372-a567-0e02b2c3d479 "Buy milk" -p high -d 2025-01-20
todocli todo create f47ac10b-58cc-4372-a567-0e02b2c3d479 "Buy eggs" -p medium
todocli todo create f47ac10b-58cc-4372-a567-0e02b2c3d479 "Buy bread" -p low

# 4. List all todos
todocli todo ls f47ac10b-58cc-4372-a567-0e02b2c3d479

# 5. Mark first todo as complete
todocli todo update f47ac10b-58cc-4372-a567-0e02b2c3d479 <todo-id> --completed

# 6. View all lists
todocli list ls
```

### Using with OCI Instance

```bash
# Configure for OCI instance
todocli config set --api-url https://192.18.159.108:8443/api/v1
todocli config set --insecure  # For self-signed cert

# Login
todocli auth login user@example.com

# Now all commands work with OCI instance
todocli list ls
```

### Script Integration

```bash
#!/bin/bash
# Create a shopping list and add items from a file

LIST_ID=$(todocli list create "Shopping" --json | jq -r '.id')

while IFS= read -r item; do
  todocli todo create "$LIST_ID" "$item" -p medium
done < shopping_items.txt

echo "Created list: $LIST_ID"
todocli todo ls "$LIST_ID"
```

### JSON Output for Automation

```bash
# Get all lists as JSON
todocli list ls --json | jq '.[].name'

# Get todos and filter by priority
todocli todo ls <list-id> --json | jq '.[] | select(.priority == "high")'

# Count incomplete todos
todocli todo ls <list-id> --json | jq '[.[] | select(.completed == false)] | length'
```

## Troubleshooting

### TLS Certificate Errors

If you encounter certificate errors with self-signed certs:

```bash
# Use --insecure flag
todocli --insecure auth login user@example.com

# Or set in config
todocli config set --insecure
```

### Token Expired

If your access token expires (default 15 minutes), just login again:

```bash
todocli auth login
```

Automatic token refresh will be added in a future version.

### Reset Configuration

```bash
# View config file location
todocli config path

# Delete config file to reset
rm $(todocli config path)
```

### Connection Refused

Check that:
1. API server is running
2. API URL in config is correct
3. Port is accessible (firewall rules)

```bash
# Test connectivity
curl -k https://192.18.159.108:8443/health

# Check config
todocli config show
```

## Comparison with Bash CLI

The Go CLI offers several advantages over the bash script:

| Feature | Bash Script | Go CLI |
|---------|-------------|--------|
| Type safety | ❌ | ✅ |
| Error handling | Basic | Comprehensive |
| Output formatting | JSON only | Tables + JSON |
| Tab completion | ❌ | ✅ (via cobra) |
| Parameter validation | ❌ | ✅ |
| Password masking | ❌ | ✅ |
| All API endpoints | Partial | Complete |
| Offline help | Basic | Rich, contextual |
| Dependencies | curl, jq | Single binary |

## Shell Completion

Generate shell completion scripts:

```bash
# Bash
todocli completion bash > /etc/bash_completion.d/todocli

# Zsh
todocli completion zsh > "${fpath[1]}/_todocli"

# Fish
todocli completion fish > ~/.config/fish/completions/todocli.fish

# PowerShell
todocli completion powershell > todocli.ps1
```

## Building for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o todocli-linux cmd/todocli/*.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o todocli-macos cmd/todocli/*.go

# Windows
GOOS=windows GOARCH=amd64 go build -o todocli.exe cmd/todocli/*.go

# ARM (e.g., Oracle Cloud ARM instances)
GOOS=linux GOARCH=arm64 go build -o todocli-arm64 cmd/todocli/*.go
```

## Security Best Practices

1. **Config File Permissions**: The CLI automatically sets config file permissions to 0600 (owner read/write only)
2. **Password Input**: Passwords are never echoed to the terminal
3. **Token Storage**: Tokens are stored locally; never commit config file to git
4. **TLS Verification**: Only use `--insecure` in development with self-signed certs

Add to `.gitignore`:
```
.todolist-cli.json
todocli
```

## Advanced Usage

### Environment Variable Override

You can override config with environment variables:

```bash
# Override API URL for one command
API_BASE=https://prod.example.com/api/v1 todocli list ls
```

(Note: This requires code modification to support environment variables)

### Pipeline Integration

```bash
# Export todos to CSV
todocli todo ls <list-id> --json | \
  jq -r '.[] | [.id, .description, .priority, .completed] | @csv' > todos.csv

# Import todos from JSON
cat todos.json | jq -r '.[]' | while read desc; do
  todocli todo create <list-id> "$desc"
done
```

## Support

- **Issues**: https://github.com/yourorg/todolist-api/issues
- **Documentation**: See [docs/](../docs/) directory
- **API Documentation**: https://192.18.159.108:8443/swagger/index.html

---

**Version**: 1.0
**Last Updated**: 2025-11-26
