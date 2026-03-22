#!/usr/bin/env bash

# Read JSON input from stdin
input=$(cat)

# Extract tool name and command using jq
tool_name=$(echo "$input" | jq -r '.tool_name')
command=$(echo "$input" | jq -r '.tool_input.command // ""')

# Only process run_shell_command
if [[ "$tool_name" == "run_shell_command" ]]; then
  # List of common commands used to read file contents
  read_commands=("cat " "less " "more " "head " "tail ")
  
  blocked=false
  matched_cmd=""
  
  for cmd in "${read_commands[@]}"; do
    # Check if the command string contains one of the read commands
    # We use a simple check, this could be made more robust with regex
    if [[ "$command" == *"$cmd"* ]]; then
        # Exception for git commands and pipes
        if [[ "$command" != *"git "* ]] && [[ "$command" != *"| "* ]]; then
            blocked=true
            matched_cmd="$cmd"
            break
        fi
    fi
  done

  if [[ "$blocked" == true ]]; then
    # Log to stderr for debugging
    echo "Hook: Blocked shell command '$command' attempting to use '$matched_cmd' to read files" >&2

    # Return a JSON denial instructing the agent to use mcp_neko_read_file
    cat <<EOF
{
  "decision": "deny",
  "reason": "Optimization Hook: Using standard shell commands (like '$matched_cmd') to read files is inefficient and breaks semantic awareness. You MUST use the 'mcp_neko_read_file' tool instead to read file contents within this project.",
  "systemMessage": "🛑 Blocked raw file read via shell"
}
EOF
    exit 0
  fi
fi

# Default: Allow the tool to proceed
echo '{"decision": "allow"}'
exit 0
