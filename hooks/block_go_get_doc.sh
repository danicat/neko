#!/usr/bin/env bash

# Read JSON input from stdin
input=$(cat)

# Extract tool name and command using jq
tool_name=$(echo "$input" | jq -r '.tool_name')
command=$(echo "$input" | jq -r '.tool_input.command // ""')

# Only process run_shell_command
if [[ "$tool_name" == "run_shell_command" ]]; then
  
  blocked=false
  matched_cmd=""
  reason_msg=""
  
  if [[ "$command" == *"go get"* ]]; then
      blocked=true
      matched_cmd="go get"
      reason_msg="Optimization Hook: Using 'go get' directly via shell is inefficient. You MUST use the 'mcp_neko_add_dependencies' tool instead to install dependencies, as it automatically fetches and returns the package's API documentation for you to use immediately."
  elif [[ "$command" == *"go doc"* ]]; then
      blocked=true
      matched_cmd="go doc"
      reason_msg="Optimization Hook: Using 'go doc' via shell is hard to read and parse. You MUST use the 'mcp_neko_read_docs' tool instead, which provides properly formatted markdown documentation for standard library and third-party modules."
  fi

  if [[ "$blocked" == true ]]; then
    # Log to stderr for debugging
    echo "Hook: Blocked shell command '$command' attempting to use '$matched_cmd'" >&2

    # Return a JSON denial
    cat <<EOF
{
  "decision": "deny",
  "reason": "$reason_msg",
  "systemMessage": "🛑 Blocked raw $matched_cmd via shell"
}
EOF
    exit 0
  fi
fi

# Default: Allow the tool to proceed
echo '{"decision": "allow"}'
exit 0
