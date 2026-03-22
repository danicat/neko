#!/usr/bin/env bash

# Read JSON input from stdin
input=$(cat)

# Extract tool name and command using jq
tool_name=$(echo "$input" | jq -r '.tool_name')
command=$(echo "$input" | jq -r '.tool_input.command // ""')

# Only process run_shell_command
if [[ "$tool_name" == "run_shell_command" ]]; then
  
  # List of go commands to block
  go_commands=("go build" "go test")
  
  blocked=false
  matched_cmd=""
  
  for cmd in "${go_commands[@]}"; do
    if [[ "$command" == *"$cmd"* ]]; then
        blocked=true
        matched_cmd="$cmd"
        break
    fi
  done

  if [[ "$blocked" == true ]]; then
    # Log to stderr for debugging
    echo "Hook: Blocked shell command '$command' attempting to use '$matched_cmd'" >&2

    # Return a JSON denial instructing the agent to use mcp_neko_build
    cat <<EOF
{
  "decision": "deny",
  "reason": "Optimization Hook: Using '$matched_cmd' directly via shell bypasses the project quality gates. You MUST use the 'mcp_neko_build' tool instead to build or test the project, as it guarantees synchronization with the language server and runs modernization checks.",
  "systemMessage": "🛑 Blocked raw $matched_cmd via shell"
}
EOF
    exit 0
  fi
fi

# Default: Allow the tool to proceed
echo '{"decision": "allow"}'
exit 0
